package outputredis

import (
	"reflect"
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

	outchan := plugin.Get(reflect.TypeOf(make(utils.OutChan))).
		Interface().(utils.OutChan)
	outchan <- utils.LogEvent{
		Timestamp: time.Now(),
		Message:   "new message",
		Extra: map[string]interface{}{
			"name": "tuhuayuan",
		},
	}
	plugin.StopOutputs()
}
