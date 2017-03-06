package queue

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func NewTestLogger() *logrus.Logger {
	return &logrus.Logger{
		Out:       os.Stdout,
		Formatter: &logrus.TextFormatter{},
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
}

// 测试简单的入列出列
func Test_DiskQueue(t *testing.T) {
	logger := NewTestLogger()

	dqName := "test_disk_queue" + strconv.Itoa(int(time.Now().Unix()))
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("logagent-test-%d", time.Now().UnixNano()))
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dq := New(dqName, tmpDir, 1024, 4, 1<<10, 2500, 2*time.Second, logger)
	defer dq.Close()

	assert.NotNil(t, dq)
	assert.Equal(t, int64(0), dq.Depth())

	msg := []byte("test")
	err = dq.Put(msg)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), dq.Depth())

	msgOut := <-dq.ReadChan()
	assert.Equal(t, msg, msgOut)
}

type LogEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message"`
	Tags      []string               `json:"tags,omitempty"`
	Extra     map[string]interface{} `json:"-"`
}

// 测试一下LogEvent存取，还有测试下超过单文件大小
func Test_MutilFile(t *testing.T) {
	logger := NewTestLogger()

	dqName := "test_disk_queue_mutil" + strconv.Itoa(int(time.Now().Unix()))
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("logagent-test-%d", time.Now().UnixNano()))
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	var dataBuf bytes.Buffer
	enc := gob.NewEncoder(&dataBuf)
	dec := gob.NewDecoder(&dataBuf)

	str1 := "Hello Queue"
	str2 := []string{"1", "2", "3"}
	// 文件指针会报错
	// f1, _:= os.Open(TempDir)

	eventSend := LogEvent{
		Timestamp: time.Now(),
		Message:   "new message",
		Extra: map[string]interface{}{
			"name": "tuhuayuan",
			"str1": &str1,
			"str2": str2,
			// "f1": f1,
		},
	}
	err = enc.Encode(eventSend)
	assert.NoError(t, err)

	dq := New(dqName, tmpDir, 1024, 4, 1<<10, 2500, 2*time.Second, logger)
	defer dq.Close()

	for i := 0; i < 16; i++ {
		err = dq.Put(dataBuf.Bytes())
		assert.NoError(t, err)
	}
	assert.Equal(t, int64((dataBuf.Len()+4)*16/1024), dq.(*diskQueue).writeFileNum)

	dataBuf.Reset()
	dataBuf.Write(<-dq.ReadChan())
	var eventReciv LogEvent
	err = dec.Decode(&eventReciv)
	assert.NoError(t, err)

	fmt.Println(eventReciv)
}

func Test_QueueDepth(t *testing.T) {
	logger := NewTestLogger()

	dqName := "test_disk_queue_mutil" + strconv.Itoa(int(time.Now().Unix()))
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("logagent-test-%d", time.Now().UnixNano()))
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	msg := bytes.Repeat([]byte{0}, 10)
	dq := New(dqName, tmpDir, 100, 0, 1<<10, 2500, 2*time.Second, logger)
	defer dq.Close()
	assert.Equal(t, int64(0), dq.Depth())

	for i := 0; i < 100; i++ {
		err = dq.Put(msg)
		assert.NoError(t, err)
	}

	for i := 0; i < 3; i++ {
		<-dq.ReadChan()
	}
	// 异步读取所以要等待一下
	for {
		if dq.Depth() == 97 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.Equal(t, int64(97), dq.Depth())

	// 测试清空
	dq.Empty()
	_, err = os.Open(dq.(*diskQueue).metaDataFileName())
	assert.True(t, os.IsNotExist(err))
	err = dq.Put(msg)
	assert.NoError(t, err)
	assert.True(t, dq.(*diskQueue).writePos > dq.(*diskQueue).readPos)
}

func Test_Corruption(t *testing.T) {
	logger := NewTestLogger()

	dqName := "test_disk_queue_corruption" + strconv.Itoa(int(time.Now().Unix()))
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("logagent-test-%d", time.Now().UnixNano()))
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dq := New(dqName, tmpDir, 1000, 10, 1<<10, 2500, 1*time.Second, logger)
	defer dq.Close()

	msg := make([]byte, 127-4)
	// 一共产生4个文件, 8消息一个文件，最后一个文件只有一个消息
	for i := 0; i < 25; i++ {
		dq.Put(msg)
	}
	assert.Equal(t, int64(25), dq.Depth())

	// 截断第二个文件, 留下三个合法的消息
	f2 := dq.(*diskQueue).fileName(1)
	os.Truncate(f2, 500)
	// 留下最后一个消息
	for i := 0; i < 19; i++ {
		assert.Equal(t, msg, <-dq.ReadChan())
	}
	// 我们把要写入的文件截断
	f3 := dq.(*diskQueue).fileName(3)
	os.Truncate(f3, 100)
	// 等待队列发现错误
	for {
		if dq.(*diskQueue).nextReadFileNum == 4 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	// 写入一个消息,创建第5个文件
	err = dq.Put(msg)
	assert.NoError(t, err)
	assert.Equal(t, msg, <-dq.ReadChan())
	// 直接写入一个非法数据
	_, err = dq.(*diskQueue).writeFile.Write([]byte{0, 0, 0, 0})
	assert.NoError(t, err)
	// 触发ioloop去读新消息
	dq.Put(msg)
	// 等待队列发现错误
	for {
		if dq.(*diskQueue).nextReadFileNum == 5 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	// 一切回归正常
	dq.Put(msg)
	assert.Equal(t, msg, <-dq.ReadChan())
}

func Test_Concurrency(t *testing.T) {
	var waiter sync.WaitGroup
	logger := NewTestLogger()

	dqName := "test_disk_queue_corruption" + strconv.Itoa(int(time.Now().Unix()))
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("logagent-test-%d", time.Now().UnixNano()))
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dq := New(dqName, tmpDir, 1024000, 0, 1<<10, 2500, 1*time.Second, logger)
	msg := bytes.Repeat([]byte{0}, 64)

	numWriters := 4
	numReader := 4
	readerExitChan := make(chan bool)
	writerExitChan := make(chan bool)

	var depth int64
	for i := 0; i < numWriters; i++ {
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			for {
				time.Sleep(100000 * time.Nanosecond)
				select {
				case <-writerExitChan:
					return
				default:
					err := dq.Put(msg)
					if err == nil {
						atomic.AddInt64(&depth, 1)
					}
				}
			}
		}()
	}

	time.Sleep(1 * time.Second)
	dq.Close()

	logger.Info("Closing")
	close(writerExitChan)
	waiter.Wait()
	logger.Info("Writer all stopped")

	dq = New(dqName, tmpDir, 1024000, 0, 1<<10, 2500, 1*time.Second, logger)
	var read int64
	for i := 0; i < numReader; i++ {
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			for {
				select {
				case <-readerExitChan:
					return
				case readMsg := <-dq.ReadChan():
					assert.Equal(t, msg, readMsg)
					atomic.AddInt64(&read, 1)
				}
			}
		}()
	}

	// 等待全部读完
	for {
		if dq.Depth() == 0 {
			close(readerExitChan)
			waiter.Wait()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	assert.Equal(t, read, depth)
	dq.Close()
}

func Test_Peek(t *testing.T) {
	logger := NewTestLogger()

	dqName := "test_disk_queue_peek" + strconv.Itoa(int(time.Now().Unix()))
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("logagent-test-%d", time.Now().UnixNano()))
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dq := New(dqName, tmpDir, 1024000, 0, 1<<10, 2500, 1*time.Second, logger)
	defer dq.Close()
	msg := bytes.Repeat([]byte{1}, 64)

	err = dq.Put(msg)
	assert.NoError(t, err)

	assert.Equal(t, msg, <-dq.PeekChan())
	assert.Equal(t, msg, <-dq.PeekChan())
	assert.Equal(t, msg, <-dq.ReadChan())
	for {
		if dq.Depth() == 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	assert.Equal(t, int64(0), dq.Depth())
}
