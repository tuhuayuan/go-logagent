package inputhttp

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tuhuayuan/go-logagent/utils"
)

func init() {
	utils.RegistInputHandler(PluginName, InitHandler)
}

func Test_Run(t *testing.T) {
	config, err := utils.LoadFromString(`
	{
		"input": [
			{
				"type": "http",
				"host": "0.0.0.0:3000",
				"url_path": "/replay",
				"methods": ["POST", "Put"]
			}
		]
	}
	`)
	assert.NoError(t, err)
	err = config.RunInputs()
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)
	client := &http.Client{}
	reader := bytes.NewReader([]byte(`
	{
		"id": "ddsfsdf",
		"data": "dfsafdsafas"
	}
	`))
	_, err = client.Post("http://localhost:3000", "application/json", reader)
	assert.NoError(t, err)
	config.StopInputs()
}
