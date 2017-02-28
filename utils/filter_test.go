package utils

import (
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
)

type TestFilterPlugin struct {
	FilterPluginConfig

	Name string `json:"name"`
}

var (
	testFilterPlugin = &TestFilterPlugin{}
	tfpIndex         = 0
)

func InitTestFilterPlugin(part *ConfigPart) *TestFilterPlugin {
	ReflectConfigPart(part, &testFilterPlugin)
	return testFilterPlugin
}

func (tp *TestFilterPlugin) Process(before LogEvent) LogEvent {
	before.Message += " tuhuayuan"
	return before
}

func fakeInput(input InChan) {
	input <- LogEvent{
		Timestamp: time.Now(),
		Message:   "hello",
		Tags:      []string{},
		Extra:     map[string]interface{}{"index": tfpIndex},
	}
	tfpIndex++
}

func fakeOutput(output OutChan) LogEvent {
	return <-output
}

func Test_RunFilters(t *testing.T) {
	RegistFilterHandler("test", InitTestFilterPlugin)
	config, err := LoadFromString(`
	{
		"filter": [{
			"type": "test",
			"name": "tuhuayuan"
		}]
	}
	`)
	assert.NoError(t, err, "load config from string return an error")
	err = config.RunFilters()
	assert.NoError(t, err)
	config.Injector.Invoke(fakeInput)
	config.Injector.Invoke(fakeInput)
	config.Injector.Map(t)
	v1, err := config.Injector.Invoke(fakeOutput)
	e1 := v1[0].Interface().(LogEvent)
	v2, err := config.Injector.Invoke(fakeOutput)
	e2 := v2[0].Interface().(LogEvent)
	assert.Equal(t, e2.Message, "hello tuhuayuan", "filter process error.")
	assert.True(t, e1.Extra["index"].(int) < e2.Extra["index"].(int))
	err = config.StopFilters()
	assert.NoError(t, err)
}
