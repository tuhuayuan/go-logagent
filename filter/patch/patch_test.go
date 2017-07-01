package patchfilter

import (
	"testing"

	"github.com/tuhuayuan/go-logagent/utils"

	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	utils.RegistFilterHandler(PluginName, InitHandler)
}

func Test_Event(t *testing.T) {
	conf, err := utils.LoadFromString(`{
		"filter": [{
			"type": "patch",
			"key": "name",
			"value": "tuhuayuan"
		}]
	}`)
	assert.NoError(t, err)
	plugin, err := InitHandler(&conf.FilterPart[0])
	assert.NoError(t, err)

	ev := utils.LogEvent{
		Timestamp: time.Now(),
		Message:   "",
	}

	ev = plugin.Process(ev)
	assert.Equal(t, "tuhuayuan", ev.Extra["name"])
}
