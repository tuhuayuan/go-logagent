package utils

import (
	"testing"

	"strings"

	"github.com/stretchr/testify/assert"
)

var endpoints = []string{
	"http://172.17.0.1:7001",
	"http://172.17.0.1:7002",
}

func Test_CmdLocalPath(t *testing.T) {
	t.Run("Path not exitst.", func(t *testing.T) {
		_, err := CmdLocalPath("")
		assert.Error(t, err)
	})

	t.Run("Templates path.", func(t *testing.T) {
		_, err := CmdLocalPath("../templates")
		assert.NoError(t, err)
	})

	t.Run("Template file.", func(t *testing.T) {
		_, err := CmdLocalPath("../templates/default.json")
		assert.NoError(t, err)
	})
}

func Test_CmdEtcdHost(t *testing.T) {
	_, err := CmdEtcdHost(strings.Join(endpoints, "; "))
	assert.NoError(t, err)
}

func Test_cmdFileList(t *testing.T) {
	files, err := cmdFileList("../templates", "json")
	assert.NotEmpty(t, files)
	assert.NoError(t, err)
}

func Test_cmdEndpointList(t *testing.T) {

	eps, err := cmdEndpointList(strings.Join(endpoints, "; "), ";")
	assert.NoError(t, err)
	assert.Len(t, eps, 2)
}
