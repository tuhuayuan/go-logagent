package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"zonst/tuhuayuan/logagent/utils"

	"github.com/coreos/etcd/client"
	"github.com/fsnotify/fsnotify"
)

func runSentinel() int {
	cmd := getAgentCmd()
	err := cmd.Start()
	if err != nil {
		utils.Logger.Errorf("start agent error %s", err)
		return -1
	}
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	watchChan := make(chan bool, 1)
	if *etcdHosts != "" {
		go watchEtcd(watchChan)
	} else if *configDir != "" {
		go watchDir(watchChan)
	}

	for {
		select {
		case sig := <-signalChan:
			cmd.Process.Signal(sig)
			_, err := cmd.Process.Wait()
			if err != nil {
				utils.Logger.Warnf("Subprocess exit error %s", err)
				return -1
			}
			return 0
		case <-watchChan:
			utils.Logger.Warn("Config changed, agent restarting.")
			cmd.Process.Signal(syscall.SIGINT)
			_, err := cmd.Process.Wait()
			if err != nil {
				utils.Logger.Warnf("Subprocess exit stop error %s", err)
				// kill immediately
				cmd.Process.Kill()
			}
			cmd = getAgentCmd()
			err = cmd.Start()
			if err != nil {
				utils.Logger.Fatalf("Subprocess restart error %s", err)
				return -1
			}
		}
	}
}

func getAgentCmd() *exec.Cmd {
	var agentArgs []string
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-sentinel") {
			agentArgs = append(agentArgs, arg)
		}
	}
	cmd := exec.Command(os.Args[0], agentArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// watching etcd config change.
func watchEtcd(changed chan bool) {
	cfg := client.Config{
		Endpoints: getEtcdList(),
		Transport: client.DefaultTransport,
	}
	c, err := client.New(cfg)
	if err != nil {
		utils.Logger.Errorf("Watching etcd client error %s", err)
		return
	}
	api := client.NewKeysAPI(c)
	w := api.Watcher(getEtcdPath(), nil)
	for {
		_, err := w.Next(context.Background())
		if err != nil {
			utils.Logger.Errorf("Watching etcd client error %s", err)
			return
		}
		changed <- true
	}
}

// watching config dir change.
func watchDir(event chan bool) {
	var (
		dir string
		fi  os.FileInfo
	)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		utils.Logger.Errorf("Watching dir error %s", err)
		return
	}

	if dir, err = filepath.EvalSymlinks(*configDir); err != nil {
		utils.Logger.Errorf("Get symlinks failed: %s error %s", dir, err)
		return
	}
	if fi, err = os.Stat(dir); err != nil {
		utils.Logger.Errorf("Get file status %s error %s", dir, err)
		return
	}
	if !fi.IsDir() {
		utils.Logger.Errorf("not a directory %s", fi.Name())
		return
	}
	err = watcher.Add(dir)
	if err != nil {
		utils.Logger.Errorf("Watching add dir error %s", err)
		return
	}
	// watch any change
	for {
		select {
		case <-watcher.Events:
			event <- true
		}
	}

}
