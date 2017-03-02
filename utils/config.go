package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"

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
}

// InChan input channel
type InChan chan LogEvent

// OutChan output channel
type OutChan chan LogEvent

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

// Invoke invoke f with reflected value.
func (c *TypePluginConfig) Invoke(f interface{}) (refvs []reflect.Value, err error) {
	if refvs, err = c.Injector.Invoke(f); err != nil {
		return
	}
	err = checkError(refvs)
	return
}

// LoadFromDir load from file path.
func LoadFromDir(path string) (configs []Config, err error) {
	fi, err := os.Stat(path)
	if err != nil {
		return
	}
	if !fi.IsDir() {
		err = errors.New("config path is not a directory")
		return

	}
	flist, err := FileList(path, "json")
	if err != nil {
		err = errors.New("config path error " + err.Error())
		return
	}
	for _, f := range flist {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			Logger.Warnf("Read config file error %s", err)
			continue
		}
		config, err := LoadFromData(data)
		if err != nil {
			Logger.Warnf("Load config from date  %s", err)
			continue
		}
		configs = append(configs, config)
	}
	return
}

// LoadFromString load from golang string.
func LoadFromString(text string) (config Config, err error) {
	return LoadFromData([]byte(text))
}

// LoadFromNode load config from etcd node.
func LoadFromNode(endpoints []string, root string) (configs []Config, err error) {
	cfg := client.Config{
		Endpoints: endpoints,
		Transport: client.DefaultTransport,
	}
	c, err := client.New(cfg)
	if err != nil {
		Logger.Errorln("Etcd client error.")
		return
	}
	api := client.NewKeysAPI(c)
	resp, err := api.Get(context.Background(), root, nil)
	if err != nil || !resp.Node.Dir {
		return
	}
	for _, n := range resp.Node.Nodes {
		conf, err := LoadFromString(n.Value)
		if err != nil {
			Logger.Warnln("LoadFromNode found a error config node.")
			continue
		}
		configs = append(configs, conf)
	}
	return configs, nil
}

// LoadFromData do the actual work.
func LoadFromData(data []byte) (config Config, err error) {
	if data, err = cleanComments(data); err != nil {
		return
	}

	if err = json.Unmarshal(data, &config); err != nil {
		Logger.Errorf("LoadFromDate json unmarshal error %s", err)
		return
	}

	config.Injector = inject.New()
	config.Map(Logger)

	inchan := make(InChan, 100)
	outchan := make(OutChan, 100)

	config.Map(inchan)
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

// InvokeSimple do invoke on injector.
func (c *Config) InvokeSimple(arg interface{}) (err error) {
	refvs, err := c.Injector.Invoke(arg)
	if err != nil {
		return
	}
	err = checkError(refvs)
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
