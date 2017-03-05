package utils

import (
	"testing"
	"time"
	"zonst/tuhuayuan/logagent/queue"

	"bytes"
	"encoding/gob"

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

func (plugin *TestFilterPlugin) Output(ev LogEvent) error {
	testOutputChan <- ev
	return nil
}
func (plugin *TestFilterPlugin) Process(ev LogEvent) LogEvent {
	return ev
}

func Test_RunFilters(t *testing.T) {
	RegistFilterHandler("test_filter", InitTestFilterPlugin)
	config, err := LoadFromString(`
	{
		"filter": [{
			"type": "test_filter"
		}]
	}
	`)
	assert.NoError(t, err)
	dq := queue.New(config.Name, config.DataPath,
		1024*1024*1024,
		0,
		1024*1024*10,
		1024,
		1*time.Second,
		Logger)
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	config.Map(&buf)
	config.Map(enc)
	config.Map(dec)
	config.Map(dq)
	config.MapTo(testFilterPlugin, (*OutputChannel)(nil))

	err = config.RunFilters()
	assert.NoError(t, err)

	enc.Encode(LogEvent{
		Message: "test",
	})
	dq.Put(buf.Bytes())
	assert.Equal(t, (<-testOutputChan).Message, "test")
	err = config.StopFilters()
	assert.NoError(t, err)
}
