package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestOutputPlugin struct {
	OutputPluginConfig
}

var (
	testOutputPluin = &TestOutputPlugin{}
	outputChan      = make(chan LogEvent, 1)
)

func (plugin *TestOutputPlugin) Process(ev LogEvent) (err error) {
	fmt.Println(ev)
	outputChan <- ev
	return
}

func (plugin *TestOutputPlugin) Stop() {
}

func InitTestOutputPlugin(config *ConfigPart) *TestOutputPlugin {
	ReflectConfigPart(config, testOutputPluin)
	return testOutputPluin
}

func Test_RunOutputs(t *testing.T) {
	RegistOutputHandler("test_output", InitTestOutputPlugin)
	config, err := LoadFromString(`
	{
		"output": [{
			"type": "test_output"
		}]
	}
	`)
	assert.NoError(t, err)
	err = config.RunOutputs()
	assert.NoError(t, err)
	_, err = testOutputPluin.Invoke(func(output OutputChannel) {
		fmt.Println(output)
		output.Output(LogEvent{})
	})
	assert.NoError(t, err)
	<-outputChan
	err = config.StopOutputs()
	assert.NoError(t, err)
}
