package outputredis

import (
	"testing"

	"zonst/tuhuayuan/logagent/utils"

	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

func Test_All(t *testing.T) {
	plugin, err := utils.LoadFromString(`{
		"output": [{
			"type": "redis",
	    	"key": "log",
			"db": 1,
	        "host": "127.0.0.1:6379",
	        "data_type": "list",
	        "timeout": 5
		}]
	}`)

	err = plugin.RunOutputs()
	assert.NoError(t, err)

	ev := utils.LogEvent{
		Timestamp: time.Now(),
		Message:   "new message",
		Extra: map[string]interface{}{
			"name": "tuhuayuan",
		},
	}
	_, err = plugin.Invoke(func(outChan utils.OutputChannel) {
		err = outChan.Output(ev)
	})
	assert.NoError(t, err)
	err = plugin.StopOutputs()
	assert.NoError(t, err)
}
