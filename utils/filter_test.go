package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestFilterPlugin struct {
	FilterPluginConfig
}

var (
	testFilterPlugin = &TestFilterPlugin{}
	testOutputChan   = make(chan LogEvent, 1)
)

func InitTestFilterPlugin(part *ConfigPart) *TestFilterPlugin {
	ReflectConfigPart(part, testFilterPlugin)
	return testFilterPlugin
}

func (plugin *TestFilterPlugin) Output(ev LogEvent) (err error) {
	testOutputChan <- ev
	return
}

func (plugin *TestFilterPlugin) Process(ev LogEvent) LogEvent {
	ev.Message = ev.Message + " filted"
	return ev
}

func Test_RunFilters(t *testing.T) {
	RegistFilterHandler("test_filter", InitTestFilterPlugin)
	plugin, err := LoadFromString(`
	{
		"filter": [{
			"type": "test_filter"
		}]
	}
	`)
	assert.NoError(t, err)
	err = plugin.RunFilters()
	assert.NoError(t, err)
	plugin.MapTo(testFilterPlugin, (*OutputChannel)(nil))
	_, err = plugin.Invoke(func(inChan InputChannel) {
		err = inChan.Input(LogEvent{
			Message: "test",
		})
	})
	fmt.Println(<-testOutputChan)
	assert.NoError(t, err)

	err = plugin.StopFilters()
	assert.NoError(t, err)
}
