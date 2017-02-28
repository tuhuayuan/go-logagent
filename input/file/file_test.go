package fileinput

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"bufio"
	"bytes"
	"sync"
	"zonst/tuhuayuan/logagent/utils"
)

var (
	sincdb = "../../tmp/since/sincedb"
	config = `{
		"input": [{
			"type": "file",
			"dirspath": ["../../tmp/log"],
			"sincepath": "../../tmp/since/sincedb",
			"follow": false
		}]
	}`
	tempDir    = "../../tmp/log"
	tempFile   = "../../tmp/log/test.log"
	golangFile = "../../tmp/log/golang.log"
)

func init() {
	utils.RegistInputHandler(PluginName, InitHandler)
}

func Test_waitWatchEvent(t *testing.T) {
	okChan := make(chan bool)

	go func() {
		_, err := waitWatchEvent(tempFile, 0, &sync.RWMutex{})
		assert.NoError(t, err)
		err = os.Remove(tempFile)
		okChan <- true
	}()

	time.Sleep(time.Millisecond * 100)
	f, err := os.Create(tempFile)
	assert.NoError(t, err)
	f.Close()
	assert.True(t, <-okChan)
}

func Test_readLine(t *testing.T) {
	f, err := os.OpenFile(golangFile, os.O_RDONLY, 0)
	assert.NoError(t, err)
	defer f.Close()

	buffReader := bufio.NewReader(f)
	buff := &bytes.Buffer{}
	_, size, err := readLine(buffReader, buff)
	assert.NoError(t, err)
	assert.Equal(t, 148, size)
}

var inChan utils.InChan

func getInChan(in utils.InChan) {
	inChan = in
}

func Test_Run(t *testing.T) {
	os.Remove(sincdb)
	plugin, err := utils.LoadFromString(config)
	assert.NoError(t, err)
	_, err = plugin.Injector.Invoke(getInChan)
	assert.NoError(t, err)
	err = plugin.RunInputs()
	assert.NoError(t, err)

	timeout := make(chan bool)
	go func() {
		time.Sleep(2 * time.Second)
		timeout <- true
	}()
	fi, err := os.Stat("../../tmp/log/golang.log")
	fmt.Println(fi.Size())
	var event utils.LogEvent
	for {
		select {
		case <-timeout:
			assert.Equal(t, 1180, int(event.Extra["offset"].(int64)))
			return
		case event = <-inChan:
			fmt.Printf("Message %s,  size %d, offset %d \n", event.Message, event.Extra["size"], event.Extra["offset"])
		}
	}
}

func Test_StartStop(t *testing.T) {
	os.Remove("../../tmp/since/sincedb_startstop")
	config = `{
		"input": [{
			"type": "file",
			"dirspath": ["../../tmp/log"],
			"sincepath": "../../tmp/since/sincedb_startstop",
			"follow": false
		}]
	}`
	plugin, err := utils.LoadFromString(config)
	assert.NoError(t, err)
	plugin.RunInputs()
	time.Sleep(2 * time.Second)
	plugin.StopInputs()
}

func Test_Tail(t *testing.T) {
	os.Remove("../../tmp/since/sinceddb_tail")
	config = `{
		"input": [{
			"type": "file",
			"dirspath": ["../../tmp/log2"],
			"sincepath": "../../tmp/since/sinceddb_tail",
			"follow": true
		}]
	}`
	plugin, err := utils.LoadFromString(config)
	assert.NoError(t, err)
	plugin.RunInputs()
	time.Sleep(2 * time.Second)
	_, err = plugin.Injector.Invoke(getInChan)
	assert.NoError(t, err)
	f, err := os.OpenFile("../../tmp/log2/append.log", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	assert.NoError(t, err)
	defer f.Close()
	w := bufio.NewWriter(f)
	var c int
	c, err = w.WriteString("test line1 \n")
	fmt.Println(c, err)
	c, err = w.WriteString("test line2 \r\n")
	fmt.Println(c, err)
	err = w.Flush()
	assert.NoError(t, err)

	timeout := make(chan bool)
	go func() {
		time.Sleep(2 * time.Second)
		timeout <- true
	}()
	var event utils.LogEvent
	for {
		select {
		case <-timeout:
			return
		case event = <-inChan:
			fmt.Printf("Message %s,  size %d, offset %d \n", event.Message, event.Extra["size"], event.Extra["offset"])
		}
	}
}
