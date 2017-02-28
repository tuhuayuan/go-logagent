package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"regexp"

	"github.com/codegangsta/inject"
	"github.com/coreos/etcd/client"
)

// Defaultconfig hardcode default config
const Defaultconfig = `
{
    "input": [
        {
            "type": "stdin"
        }
    ],
    "filter": [
        {
            "type": "zeus",
            "key": "foo",
            "value": "bar"
        }
    ],
    "output": [
        {
            "type": "stdout"
        }
    ]
}
`

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

// ConfigPart pary of a config node (input, filter, output)
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

// LoadFromFile load from file path.
func LoadFromFile(path string) (config Config, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	return LoadFromData(data)
}

// LoadFromString load from golang string.
func LoadFromString(text string) (config Config, err error) {
	return LoadFromData([]byte(text))
}

// LoadDefaultConfig load from hardcode string.
func LoadDefaultConfig() (config Config, err error) {
	return LoadFromString(Defaultconfig)
}

// LoadFromNode load config from etcd node.
func LoadFromNode(endpoints []string) (configs []Config, err error) {
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
	key := "/zonst.org/logagent/" + agentName + "/"
	resp, err := api.Get(context.Background(), key, nil)
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
