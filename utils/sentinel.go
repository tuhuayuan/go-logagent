package utils

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/fsnotify/fsnotify"
)

// Sentinel 守护者接口
type Sentinel interface {
	Run() error
	Stop()
}

// agentSentinel 具体实现
type agentSentinel struct {
	ag           *Agent
	watchChan    chan int
	exitChan     chan int
	exitSyncChan chan int
}

// CreateSentinel 创建守护者
func (ag *Agent) CreateSentinel() (Sentinel, error) {
	agStl := &agentSentinel{
		ag:           ag,
		watchChan:    make(chan int),
		exitChan:     make(chan int),
		exitSyncChan: make(chan int),
	}
	return agStl, nil
}

// Run 运行Sentinel
func (agStl *agentSentinel) Run() (err error) {
	var (
		running = new(bool)
		changed = false

		wg sync.WaitGroup
	)
	*running = true

	// keep agent running
	go func() {
		wg.Add(1)
		defer wg.Done()
		restart := 5
		for *running {
			agStl.ag.Run()
			if *running {
				Logger.Warnf("Agent will restart in %d seconds.", restart)
				time.Sleep(5 * time.Second)
			}
		}
	}()

	// trace configs.
	if agStl.ag.EtcdHosts != "" {
		go agStl.watchEtcd()
	} else if agStl.ag.ConfigDir != "" {
		go agStl.watchDir()
	} else {
		err = errors.New("both etcdhosts and configdir is empty string")
		return
	}

	// main loop
	wg.Add(1)
	for *running {
		select {
		case <-agStl.exitChan:
			*running = false
			agStl.ag.Stop()
			wg.Done()
		case <-agStl.watchChan:
			if !changed {
				changed = true
				go func() {
					time.Sleep(5 * time.Second)
					agStl.ag.Stop()
					changed = false
				}()
			}
		}
	}
	wg.Wait()
	close(agStl.exitSyncChan)
	return
}

// Stop 停止Sentinel以及对应的Agent
func (agStl *agentSentinel) Stop() {
	close(agStl.exitChan)
	<-agStl.exitSyncChan
}

// watchEtcd 监听ETCD节点变化
func (agStl *agentSentinel) watchEtcd() {
	cfg := client.Config{
		Endpoints: getEtcdList(agStl.ag.EtcdHosts),
		Transport: client.DefaultTransport,
	}
	c, err := client.New(cfg)
	if err != nil {
		Logger.Errorf("Watching etcd client error %s", err)
		return
	}
	api := client.NewKeysAPI(c)
	w := api.Watcher(getEtcdPath(agStl.ag.Name), nil)
	for {
		_, err := w.Next(context.Background())
		if err != nil {
			Logger.Errorf("Watching etcd client error %s", err)
			return
		}
		agStl.watchChan <- 1
	}
}

// watchDir 监听本地配置文件夹变化
func (agStl *agentSentinel) watchDir() {
	var (
		dir string
		fi  os.FileInfo
	)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		Logger.Errorf("Watching dir error %s", err)
		return
	}
	if dir, err = filepath.EvalSymlinks(agStl.ag.ConfigDir); err != nil {
		Logger.Errorf("Get symlinks failed: %s error %s", dir, err)
		return
	}
	if fi, err = os.Stat(dir); err != nil {
		Logger.Errorf("Get file status %s error %s", dir, err)
		return
	}
	if !fi.IsDir() {
		Logger.Errorf("not a directory %s", fi.Name())
		return
	}
	err = watcher.Add(dir)
	if err != nil {
		Logger.Errorf("Watching add dir error %s", err)
		return
	}
	for {
		select {
		case e := <-watcher.Events:
			if e.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) > 0 {
				agStl.watchChan <- 1
			}
		}
	}

}
