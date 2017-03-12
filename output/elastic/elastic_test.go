package outputelastic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"encoding/json"
	"zonst/qipai/logagent/utils"
)

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

type GamePlayer struct {
	// 持久化字段（全兼容)
	UserID   int64  `json:"user_id"`
	NickName string `json:"nick_name"`
	Point    int    `json:"point"`
}

func Test_Run(t *testing.T) {
	config := `
    {
        "output": [
            {
                "type": "elastic",
                "hosts": ["localhost:9200"]
            }
        ]
    }
    `

	plugin, err := utils.LoadFromString(config)
	assert.NoError(t, err)
	err = plugin.RunOutputs()
	assert.NoError(t, err)
	gp := &GamePlayer{
		UserID:   100,
		NickName: "tuhuayuan",
		Point:    -11,
	}
	raw, err := json.Marshal(gp)
	ev := utils.LogEvent{
		Timestamp: time.Now(),
		Message:   string(raw),
		Extra: map[string]interface{}{
			"@elastic_indices": "${@date}.logagent.test",
			"@elastic_doctype": "test",
			"@elastic_docid":   "",
		},
	}

	_, err = plugin.Invoke(func(outChan utils.OutputChannel) {
		err = outChan.Output(ev)
	})
	assert.NoError(t, err)
	plugin.StopOutputs()
	assert.NoError(t, err)
}
