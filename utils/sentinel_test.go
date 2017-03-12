package utils

import (
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
)

func Test_watchEtcd(t *testing.T) {
}

func Test_watchDir(t *testing.T) {
}

func Test_Create(t *testing.T) {
	var (
		err error
	)
	ag := NewAgent()
	ag.Name = "test_agent"
	ag.ConfigDir = "../test/empty"
	ag.DataDir = "../tmp"

	stl, err := ag.CreateSentinel()
	assert.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Second)
		stl.Stop()
	}()
	err = stl.Run()
	assert.NoError(t, err)
}
