package utils

import (
	"context"
	"testing"

	"github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
)

func Test_LoadFromString(t *testing.T) {
	t.Run("Load good json config.", func(t *testing.T) {
		c, err := LoadFromString(`
		{
			"input": [{
				"type": "file",
				"path": "./tmp/log/log.log",
				"sincedb_path": "",
				"start_position": "beginning"
			}]
		}
		`)
		assert.NoError(t, err)
		assert.Equal(t, "file", c.InputPart[0]["type"])
	})
}

func Test_LoadFromNode(t *testing.T) {
	eps := []string{
		"http://localhost:2379",
	}
	agent := "gameserver-test-001"
	CmdAgentName(agent)
	cfg := client.Config{
		Endpoints: eps,
		Transport: client.DefaultTransport,
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)
	api := client.NewKeysAPI(c)
	key := "/zonst.org/logagent/" + agent + "/"
	api.Delete(context.Background(), key, nil)
	api.Set(context.Background(), key, Defaultconfig, nil)
	_, err = LoadFromNode(eps, key)
	assert.NoError(t, err)
}
