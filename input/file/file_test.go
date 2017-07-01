package fileinput

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tuhuayuan/go-logagent/queue"
	"github.com/tuhuayuan/go-logagent/utils"
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

func Test_Run(t *testing.T) {
	os.Remove("../../tmp/since/sincedb")
	plugin, err := utils.LoadFromString(`{
		"input": [{
			"type": "file",
			"dirspath": ["../../tmp/log"],
			"sincepath": "../../tmp/since/sincedb",
			"follow": false
		}]
	}`)
	assert.NoError(t, err)

	err = plugin.RunInputs()
	assert.NoError(t, err)
	plugin.Invoke(func(dq queue.Queue) {
		fmt.Println(<-dq.ReadChan())
	})
	plugin.Invoke(func(dq queue.Queue) {
		fmt.Println(<-dq.ReadChan())
	})
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
	time.Sleep(1 * time.Second)
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
	assert.NoError(t, err)
	f, err := os.OpenFile("../../tmp/log2/append.log", os.O_APPEND|os.O_WRONLY, 0600)
	assert.NoError(t, err)
	defer f.Close()
	time.Sleep(1 * time.Second)
	c, err := f.WriteString("test message line1 \n")
	fmt.Println(c, err)
	c, err = f.WriteString("line2\n")
	fmt.Println(c, err)
	f.Close()

	plugin.Invoke(func(dq queue.Queue) {
		fmt.Println(<-dq.ReadChan())
	})
}
