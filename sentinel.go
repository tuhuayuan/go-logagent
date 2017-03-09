package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"zonst/qipai/logagent/utils"

	"time"

	"strconv"

	"github.com/coreos/etcd/client"
	"github.com/fsnotify/fsnotify"
)

func runSentinel() int {
	var (
		exit    chan bool
		running bool
		cmdChan chan *exec.Cmd
	)
	cmdChan = make(chan *exec.Cmd, 1)
	pf, err := os.OpenFile(*pidFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		utils.Logger.Fatalf("Write pid file error %s", err)
		return -1
	}
	defer pf.Close()
	pf.WriteString(strconv.Itoa(os.Getpid()))

	// trace agent process state.
	exit = make(chan bool, 1)
	running = true
	go watchAgentProcess(exit, &running, cmdChan)

	// trace system signal.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	// trace configs.
	watchChan := make(chan bool, 1)
	if *etcdHosts != "" {
		go watchEtcd(watchChan)
	} else if *configDir != "" {
		go watchDir(watchChan)
	}

	// main loop
	for {
		current := <-cmdChan
		select {
		case sig := <-signalChan:
			running = false
			current.Process.Signal(sig)
			<-exit
			return 0
		case <-watchChan:
			utils.Logger.Warn("Config changed, agent restarting.")
			current.Process.Signal(syscall.SIGINT)
		}
	}
}

// get agent command.
func getAgentCmd() *exec.Cmd {
	agentArgs := []string{"-pid", strconv.Itoa(os.Getpid())}
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-sentinel") {
			agentArgs = append(agentArgs, arg)
		}
	}
	cmd := exec.Command(os.Args[0], agentArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd
}

// watching subprocess.
func watchAgentProcess(exit chan bool, running *bool, cmdChan chan *exec.Cmd) {
	startup := make(chan os.Signal, 1)
	signal.Notify(startup, syscall.SIGUSR1)

	for *running {
		cmd := getAgentCmd()
		err := cmd.Start()
		if err != nil {
			utils.Logger.Warnf("Agent start process error %s", err)
		}
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(5 * time.Second)
			timeout <- true
		}()
		select {
		case <-timeout:
		case <-startup:
		}
		cmdChan <- cmd
		state, err := cmd.Process.Wait()
		if err != nil || !state.Success() {
			utils.Logger.Warnf("Agent process error %q, %q", state, err)
		}
		if *running {
			utils.Logger.Warnf("Agent process will restart in %d second.", 5)
			time.Sleep(time.Duration(5) * time.Second)
		}
	}
	exit <- true
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
		case e := <-watcher.Events:
			if e.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) > 0 {
				if len(event) == 0 {
					event <- true
				}
			}
		}
	}

}
