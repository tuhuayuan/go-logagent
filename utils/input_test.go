package utils

import (
	"fmt"
	"testing"
	"zonst/tuhuayuan/logagent/queue"

	"github.com/stretchr/testify/assert"
)

var testInputPlugin = &TestInputPlugin{}

type TestInputPlugin struct {
	InputPluginConfig
	InChan InputChannel `json:"-"`
}

func (plugin *TestInputPlugin) Start() {
	plugin.InChan.Input(LogEvent{
		Message: "test",
	})
}

func (plugin *TestInputPlugin) Stop() {
}

func InitTestInputPlugin(part *ConfigPart, inchan InputChannel) *TestInputPlugin {
	ReflectConfigPart(part, &testInputPlugin)
	testInputPlugin.InChan = inchan
	return testInputPlugin
}

func Test_RunInputs(t *testing.T) {
	RegistInputHandler("test_input", InitTestInputPlugin)
	config, err := LoadFromString(`
	{
		"input": [{
			"type": "test_input"
		}]
	}
	`)
	assert.NoError(t, err)
	err = config.RunInputs()
	assert.NoError(t, err, "run inputs return an error")
	config.Invoke(func(dq queue.Queue) {
		raw := <-dq.ReadChan()
		fmt.Println(raw)
	})
	err = config.StopInputs()
}
