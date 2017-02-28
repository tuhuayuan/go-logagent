package stdininput

import (
	"testing"

	"zonst/tuhuayuan/logagent/utils"

	"github.com/stretchr/testify/assert"
)

var inChan utils.InChan

func getInChan(in utils.InChan) {
	inChan = in
}

func Test_All(t *testing.T) {
	conf, err := utils.LoadFromString(`{
		"input": [{
			"type": "stdin",
			"prefix": "input #"
		}]
	}`)
	conf.Injector.Invoke(getInChan)
	err = conf.RunInputs()
	assert.NoError(t, err)
	// test just one logevent
	<-inChan
	assert.True(t, len(inChan) > 0)
	conf.StopInputs()
}
