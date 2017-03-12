package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_RunAgentt(t *testing.T) {
	var (
		err error
	)
	ag := NewAgent()
	ag.Name = "test_agent"
	ag.ConfigDir = "../test/empty"
	ag.DataDir = "../tmp"

	go func() {
		time.Sleep(1 * time.Second)
		ag.Stop()
	}()
	if err = ag.Run(); err != nil {
		t.Fail()
	}
	assert.NoError(t, err)
}
