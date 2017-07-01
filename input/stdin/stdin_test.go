package stdininput

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tuhuayuan/go-logagent/utils"
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
