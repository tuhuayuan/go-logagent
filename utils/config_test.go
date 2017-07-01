package utils

import (
	"context"
	"testing"

	"github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
)

func Test_LoadFromDir(t *testing.T) {
	path := "../test"
	data := "../tmp/queue"
	configs, err := LoadFromDir(path, data)
	assert.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.Equal(t, "default", configs[0].Name)
}

func Test_LoadFromString(t *testing.T) {
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
}

func Test_LoadFromNode(t *testing.T) {
	eps := []string{
		"http://localhost:2379",
	}
	agent := "gameserver-test-001"

	cfg := client.Config{
		Endpoints: eps,
		Transport: client.DefaultTransport,
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)
	api := client.NewKeysAPI(c)
	key := "/" + agent + "/"
	data := "../tmp/queue"
	config := `
	{
		"name": "test-config"
	}
	`
	// clear
	api.Delete(context.Background(), key, nil)
	// set config
	api.Set(context.Background(), key, "", &client.SetOptions{Dir: true})
	api.Set(context.Background(), key+"default", config, nil)
	// load
	configs, err := LoadFromNode(eps, key, data)
	assert.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.Equal(t, "test-config", configs[0].Name)
}
