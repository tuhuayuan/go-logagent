package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"errors"

	"github.com/codegangsta/inject"
	"github.com/coreos/etcd/client"
)

// TypePlugin interface of typed plugin
type TypePlugin interface {
	SetInjector(inj inject.Injector)
	GetType() string
	Invoke(f interface{}) (refvs []reflect.Value, err error)
}

// TypePluginConfig struct of typed plugin
type TypePluginConfig struct {
	inject.Injector `json:"-"`
	Type            string `json:"type"`
}

// ConfigPart subpart of a config node (input, filter, output)
type ConfigPart map[string]interface{}

// Config struct of a config file.
type Config struct {
	inject.Injector `json:"-"`
	InputPart       []ConfigPart `json:"input"`
	FilterPart      []ConfigPart `json:"filter"`
	OutputPart      []ConfigPart `json:"output"`

	Name     string `json:"name"`
	DataPath string `json:"data_path"`
}

// InputChannel .
type InputChannel interface {
	Input(ev LogEvent) error
}

// OutputChannel .
type OutputChannel interface {
	Output(ev LogEvent) error
}
type OutChan chan LogEvent

// Check reflect invoke error
func checkError(refvs []reflect.Value) (err error) {
	for _, refv := range refvs {
		if refv.IsValid() {
			refvi := refv.Interface()
			switch refvi.(type) {
			case error:
				return refvi.(error)
			}
		}
	}
	return
}

// SetInjector set injector value.
func (c *TypePluginConfig) SetInjector(inj inject.Injector) {
	c.Injector = inj
}

// GetType get config type.
func (c *TypePluginConfig) GetType() string {
	return c.Type
}

// Invoke invoke than check and return the actual error
func (c *TypePluginConfig) Invoke(f interface{}) (refvs []reflect.Value, err error) {
	if refvs, err = c.Injector.Invoke(f); err != nil {
		return
	}
	err = checkError(refvs)
	return
}

// LoadFromDir load from file path.
// configPath string where the config files to be load
// dataPath string where the diskqueue to be store
func LoadFromDir(configPath string, dataPath string) (configs []Config, err error) {
	fi, err := os.Stat(configPath)
	if err != nil {
		return
	}
	if !fi.IsDir() {
		err = errors.New("config path is not a directory")
		return

	}
	fs, err := FileList(configPath, "json")
	if err != nil {
		err = errors.New("read config files error " + err.Error())
		return
	}
	for _, configFile := range fs {
		data, err := ioutil.ReadFile(configFile)
		if err != nil {
			Logger.Warnf("Read config file error %s", err)
			continue
		}
		configName := filepath.Base(configFile)
		configName = strings.TrimSuffix(configName, filepath.Ext(configName))
		config, err := LoadFromData(data, configName, dataPath)
		if err != nil {
			Logger.Warnf("Load config file error %s", err)
			continue
		}
		configs = append(configs, config)
	}
	return
}

// LoadFromString load from golang string.
func LoadFromString(text string) (config Config, err error) {
	configName := "config_" + strconv.Itoa(int(time.Now().Unix()))
	dataPath, _ := ioutil.TempDir("", fmt.Sprintf("config-%d", time.Now().UnixNano()))
	return LoadFromData([]byte(text), configName, dataPath)
}

// LoadFromNode load config from etcd node.
// endpoints []string
// root string  path key of the configs
// dataDir string
func LoadFromNode(endpoints []string, root string, dataDir string) (configs []Config, err error) {
	cfg := client.Config{
		Endpoints: endpoints,
		Transport: client.DefaultTransport,
	}
	c, err := client.New(cfg)
	if err != nil {
		Logger.Error("Etcd client error.")
		return
	}
	api := client.NewKeysAPI(c)
	resp, err := api.Get(context.Background(), root, nil)
	if err != nil || !resp.Node.Dir {
		Logger.Warn("Etcd node is not directory.")
		return
	}
	for _, n := range resp.Node.Nodes {
		conf, err := LoadFromData([]byte(n.Value), n.Key, dataDir)
		if err != nil {
			Logger.Warnln("LoadFromNode found a error config node.")
			continue
		}
		configs = append(configs, conf)
	}
	return configs, nil
}

// LoadFromData build config from the []byte
// data []byte config json data
// configName string name of config
// dataPath string path the diskqueue data will be
func LoadFromData(data []byte, configName string, dataPath string) (config Config, err error) {
	if data, err = cleanComments(data); err != nil {
		return
	}
	if err = json.Unmarshal(data, &config); err != nil {
		Logger.Errorf("LoadFromDate json unmarshal error %s", err)
		return
	}
	if config.Name == "" {
		config.Name = configName
	}
	if config.DataPath == "" {
		config.DataPath = dataPath
	}

	config.Injector = inject.New()

	outchan := make(OutChan)

	config.Map(Logger)
	config.Map(outchan)

	rv := reflect.ValueOf(&config)
	formatReflect(rv)

	return
}

// ReflectConfigPart reflect config.
func ReflectConfigPart(part *ConfigPart, conf interface{}) (err error) {
	data, err := json.Marshal(part)
	if err != nil {
		return
	}

	if err = json.Unmarshal(data, conf); err != nil {
		return
	}

	rv := reflect.ValueOf(conf).Elem()
	formatReflect(rv)

	return
}

// Recursive reflect json to struct.
func formatReflect(rv reflect.Value) {
	if !rv.IsValid() {
		return
	}

	switch rv.Kind() {
	case reflect.Ptr:
		if !rv.IsNil() {
			formatReflect(rv.Elem())
		}
	case reflect.Struct:
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)
			formatReflect(field)
		}
	case reflect.String:
		if !rv.CanSet() {
			return
		}
		value := rv.Interface().(string)
		value = FormatWithEnv(value)
		rv.SetString(value)
	}
}

// Supported comment formats ^\s*# and ^\s*//
func cleanComments(data []byte) (out []byte, err error) {
	reForm1 := regexp.MustCompile(`^\s*#`)
	reForm2 := regexp.MustCompile(`^\s*//`)
	data = bytes.Replace(data, []byte("\r"), []byte(""), 0) // Windows
	lines := bytes.Split(data, []byte("\n"))
	var filtered [][]byte

	for _, line := range lines {
		if reForm1.Match(line) {
			continue
		}
		if reForm2.Match(line) {
			continue
		}
		filtered = append(filtered, line)
	}

	out = bytes.Join(filtered, []byte("\n"))
	return
}

// FileList list *.suffix files in dirPath.
func FileList(dirPath string, suffix string) ([]string, error) {
	files := make([]string, 0, 128)

	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	sep := string(os.PathSeparator)
	suffix = strings.ToLower(suffix)

	for _, fi := range dir {
		if fi.IsDir() {
			// no recursive
			continue
		}
		if strings.HasSuffix(strings.ToLower(fi.Name()), suffix) {
			files = append(files, dirPath+sep+fi.Name())
		}
	}

	return files, nil
}
