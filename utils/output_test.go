package utils

import (
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
)

type TestOutputPlugin struct {
	OutputPluginConfig

	OutputFile string `json:"file"`
	Speed      int    `json:"speed"`
}

var (
	testOutput      = &TestOutputPlugin{}
	fakeDestination = make(OutChan, 2)
)

func (tp *TestOutputPlugin) Process(e LogEvent) (err error) {
	fakeDestination <- e
	return
}

func (tp *TestOutputPlugin) Stop() {

}

func InitTestOutputPlugin(config *ConfigPart) (*TestOutputPlugin, error) {
	err := ReflectConfigPart(config, testOutput)
	if err != nil {
		return nil, err
	}
	return testOutput, nil
}

func fakeSource(output OutChan) {
	output <- LogEvent{
		Timestamp: time.Now(),
		Message:   "hello",
		Tags:      []string{},
		Extra:     map[string]interface{}{"index": 0},
	}

	output <- LogEvent{
		Timestamp: time.Now(),
		Message:   "hello",
		Tags:      []string{},
		Extra:     map[string]interface{}{"index": 1},
	}
}

func Test_RunOutputs(t *testing.T) {
	config, err := LoadFromString(`
	{
		"output": [{
			"type": "test",
			"file": "memory://tmp",
			"speed": 1
		}]
	}
	`)
	assert.NoError(t, err)
	RegistOutputHandler("test", InitTestOutputPlugin)
	config.RunOutputs()
	assert.Equal(t, testOutput.Speed, 1)
	config.Injector.Invoke(fakeSource)
	for i := 0; i < 2; i++ {
		e := <-fakeDestination
		fmt.Println(e)
		assert.Equal(t, i, e.Extra["index"].(int), "message index error")
	}
	err = config.StopOutputs()
	assert.NoError(t, err)
}
