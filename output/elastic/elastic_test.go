package outputelastic

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

func Test_Init(t *testing.T) {
	config := `
    {
        "output": [
            {
                "type": "elastic",
                "hosts": ["localhost:9200"],
                "index": "${@date}.logagent",
                "doc_type": "test"
            }
        ]
    }
    `
	plugin, err := utils.LoadFromString(config)
	assert.NoError(t, err)
	plugin.RunOutputs()

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
