package grokfilter

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"zonst/tuhuayuan/logagent/utils"
)

func init() {
	utils.RegistFilterHandler(PluginName, InitHandler)
}

func Test_Process(t *testing.T) {
	conf, err := utils.LoadFromString(`{
		"filter": [{
			"type": "grok",
            "match":"\\[(?P<log_time>\\d{4}-\\d{1,2}-\\d{1,2}[^\\]]+)\\]\\[(?P<log_level>\\w{4,5})\\]\\S+(?P<log_content>.*)",
			"maxfields": 8
		}]
	}`)
	assert.NoError(t, err)
	plugin, err := InitHandler(&conf.FilterPart[0])
	assert.NoError(t, err)
	ev := utils.LogEvent{
		Timestamp: time.Now(),
		Message:   "[2017-02-21 15:53:48.881][warn][onSocketMessage][7][234][11] Frame::onSocketMessage enter. msg_id:1013,user_id:30691489, recv_time:[18]ms, frame_time[:1]ms",
		Tags:      []string{},
		Extra:     map[string]interface{}{},
	}
	ev = plugin.Process(ev)
	fmt.Println(ev)
	assert.Equal(t, "2017-02-21 15:53:48.881", ev.Extra["log_time"])
	assert.Equal(t, "warn", ev.Extra["log_level"])
}
