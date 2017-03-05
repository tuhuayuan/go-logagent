package stdininput

import (
	"testing"

	"zonst/tuhuayuan/logagent/utils"

	"github.com/stretchr/testify/assert"
)

func init() {
	utils.RegistInputHandler(PluginName, InitHandler)
}

func Test_All(t *testing.T) {
	config, err := utils.LoadFromString(`{
		"input": [{
			"type": "stdin",
			"prefix": "[test#] "
		}]
	}`)

	err = config.RunInputs()
	assert.NoError(t, err)
	config.StopInputs()
}
