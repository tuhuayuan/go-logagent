package patchfilter

import (
	"reflect"
	"testing"

	"zonst/tuhuayuan/logagent/utils"

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

	inchan := conf.Get(reflect.TypeOf(make(utils.InChan))).
		Interface().(utils.InChan)
	outchan := conf.Get(reflect.TypeOf(make(utils.OutChan))).
		Interface().(utils.OutChan)

	err = conf.RunFilters()
	assert.NoError(t, err)

	inchan <- utils.LogEvent{
		Timestamp: time.Now(),
		Message:   "",
	}

	event := <-outchan
	assert.Equal(t, "tuhuayuan", event.Extra["name"])
}
