package stdininput

import (
	"testing"

	"zonst/tuhuayuan/logagent/utils"

	"github.com/stretchr/testify/assert"
)

var inChan utils.InChan

func init() {
	utils.RegistInputHandler(PluginName, InitHandler)
}

func getInChan(in utils.InChan) {
	inChan = in
}

func Test_All(t *testing.T) {
	conf, err := utils.LoadFromString(`{
		"input": [{
			"type": "stdin",
			"prefix": "[test#] "
		}]
	}`)
	conf.Injector.Invoke(getInChan)
	err = conf.RunInputs()
	assert.NoError(t, err)
	conf.StopInputs()
}
