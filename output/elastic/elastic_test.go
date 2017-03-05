package outputelastic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"zonst/tuhuayuan/logagent/utils"
)

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

func Test_Run(t *testing.T) {
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
	plugin.StopOutputs()
	assert.NoError(t, err)
}
