package utils

import (
	"testing"

	"fmt"

	"reflect"

	"time"

	"github.com/stretchr/testify/assert"
)

var testInputPlugin = &TestInputPlugin{}

type TestInputPlugin struct {
	InputPluginConfig
	In       InChan `json:"-"`
	From     string `json:"from"`
	Username string `json:"username"`
}

func (tp *TestInputPlugin) Start() {
	fmt.Println("Plugin started.")
	testInputPlugin.In <- LogEvent{}
}

func (tp *TestInputPlugin) Stop() {
	fmt.Println("Plugin stopping")
	fmt.Printf("Flush all message %s\n", <-testInputPlugin.In)
	fmt.Println("Plugin stopped")
}

func InitTestInputPlugin(pc *ConfigPart, inchan InChan) *TestInputPlugin {
	ReflectConfigPart(pc, &testInputPlugin)
	testInputPlugin.In = inchan
	return testInputPlugin
}

func Test_NilInput(t *testing.T) {
	RegistInputHandler("test_nil", func() *TestInputPlugin {
		return nil
	})
	config, err := LoadFromString(`
	{
		"input": [{
			"type": "test_nil",
			"from": "/var/test/*.log",
			"username": "tuhuayuan"
		}]
	}
	`)
	assert.NoError(t, err)
	config.RunInputs()
	time.Sleep(1 * time.Second)
}

func Test_RunInputs(t *testing.T) {
	RegistInputHandler("test", InitTestInputPlugin)
	config, err := LoadFromString(`
	{
		"input": [{
			"type": "test",
			"from": "/var/test/*.log",
			"username": "tuhuayuan"
		}]
	}
	`)
	assert.NoError(t, err, "load from string return an error")
	err = config.RunInputs()
	assert.NoError(t, err, "run inputs return an error")
	assert.Equal(t, testInputPlugin.Injector.Get(reflect.TypeOf(make(InChan))), reflect.ValueOf(testInputPlugin.In), "InChan not correct.")
	err = config.StopInputs()
}
