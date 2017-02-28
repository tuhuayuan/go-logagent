package outputstdout

import (
	"reflect"
	"testing"
	"time"

	"zonst/tuhuayuan/logagent/utils"

	"github.com/stretchr/testify/assert"
)

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

func Test_All(t *testing.T) {
	conf, err := utils.LoadFromString(`{
		"output": [{
			"type": "stdout"
		}]
	}`)

	err = conf.RunOutputs()
	assert.NoError(t, err)

	outchan := conf.Get(reflect.TypeOf(make(utils.OutChan))).
		Interface().(utils.OutChan)
	outchan <- utils.LogEvent{
		Timestamp: time.Now(),
		Message:   "new message",
		Extra: map[string]interface{}{
			"name": "tuhuayuan",
		},
	}

	time.Sleep(2 * time.Second)
}
