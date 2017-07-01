package outputstdout

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tuhuayuan/go-logagent/utils"
)

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

func Test_All(t *testing.T) {
	plugin, err := utils.LoadFromString(`{
		"output": [{
			"type": "stdout"
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
