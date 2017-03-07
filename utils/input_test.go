package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testInputPlugin = &TestInputPlugin{}

type TestInputPlugin struct {
	InputPluginConfig
	InChan InputChannel `json:"-"`
}

func (plugin *TestInputPlugin) Start() {
	plugin.InChan.Input(LogEvent{})
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
	plugin, err := LoadFromString(`
	{
		"input": [{
			"type": "test_input"
		}]
	}
	`)

	assert.NoError(t, err)
	err = plugin.RunInputs()
	assert.NoError(t, err)
	err = plugin.StopInputs()
}
