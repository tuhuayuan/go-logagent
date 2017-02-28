package grokfilter

import (
	"fmt"
	"reflect"
	"testing"

	"zonst/tuhuayuan/logagent/utils"

	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	utils.RegistFilterHandler(PluginName, InitHandler)
}

func Test_All(t *testing.T) {
	conf, err := utils.LoadFromString(`{
		"filter": [{
			"type": "grok",
            "match":"\\[(?P<log_time>\\d{4}-\\d{1,2}-\\d{1,2}[^\\]]+)\\]\\[(?P<log_level>\\w{4,5})\\]\\S+(?P<log_content>.*)",
			"maxfields": 8
		}]
	}`)
	assert.NoError(t, err)
	err = conf.RunFilters()
	assert.NoError(t, err)

	inchan := conf.Get(reflect.TypeOf(make(utils.InChan))).
		Interface().(utils.InChan)
	outchan := conf.Get(reflect.TypeOf(make(utils.OutChan))).
		Interface().(utils.OutChan)

	inchan <- utils.LogEvent{
		Timestamp: time.Now(),
		Message:   "[2017-02-21 15:53:48.881][warn][onSocketMessage][7][234][11] Frame::onSocketMessage enter. msg_id:1013,user_id:30691489, recv_time:[18]ms, frame_time[:1]ms",
		Tags:      []string{},
		Extra:     map[string]interface{}{},
	}
	event := <-outchan
	fmt.Println(event.Extra)
	assert.Equal(t, "2017-02-21 15:53:48.881", event.Extra["log_time"])
	assert.Equal(t, "warn", event.Extra["log_level"])
}
