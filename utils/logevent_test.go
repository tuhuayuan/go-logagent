package utils

import (
	"testing"
	"time"

	"fmt"

	"encoding/json"

	"os"

	"github.com/stretchr/testify/assert"
)

func Test_AddTag(t *testing.T) {
	le := &LogEvent{
		Timestamp: time.Now(),
		Message:   "message",
		Tags:      []string{"frg", "grbhrt"},
		Extra:     make(map[string]interface{}),
	}
	le.AddTag("vftb")
	assert.Len(t, le.Tags, 3)
	le.AddTag("frg")
	assert.Len(t, le.Tags, 3)
}

func Test_GetMap(t *testing.T) {
	le := &LogEvent{
		Timestamp: time.Now(),
		Message:   "message",
		Tags:      []string{"frg", "grbhrt"},
		Extra: map[string]interface{}{
			"index":  1,
			"UserID": "tuhuayuan",
		},
	}
	jm := le.GetMap()
	d, err := json.Marshal(jm)
	assert.NoError(t, err)
	fmt.Println(string(d))
	d, err = json.MarshalIndent(jm, "", "  ")
	fmt.Println(string(d))
}

func Test_Format(t *testing.T) {
	s := "数据库://${MYSQL_HOST}:${MYSQL_PORT}, 登陆时间: ${+" + timeFormat + "}"
	os.Setenv("MYSQL_HOST", "localhost")
	os.Setenv("MYSQL_PORT", "3306")
	s = FormatWithEnv(s)
	fmt.Println(s)
	s = FormatWithTime(s)
	fmt.Println(s)
	now := time.Now()
	e := &LogEvent{
		Timestamp: now,
		Message:   "create at time  from host.",
		Tags:      []string{"frg", "grbhrt"},
		Extra: map[string]interface{}{
			"index":  1,
			"UserID": "tuhuayuan",
		},
	}
	s = e.Format("this message is create at ${@timestamp}")
	assert.Equal(t, fmt.Sprintf("this message is create at %s", now.UTC().Format(timeFormat)), s, "format error")
	fmt.Println(s)
}
