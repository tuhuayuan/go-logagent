package main

import (
	"os"
	"os/signal"
	"syscall"
	"zonst/tuhuayuan/logagent/utils"
)

// RunAgent run agent mode.
func runAgent() int {
	var (
		configs []utils.Config
		err     error
		running bool
	)
	if *etcdHosts != "" {
		configs, err = utils.LoadFromNode(getEtcdList(), getEtcdPath())
	} else {
		configs, err = utils.LoadFromDir(*configDir)
	}

	if err != nil {
		utils.Logger.Fatalf("Agent load config error %s", err)
		return -1
	}

	for _, c := range configs {
		if err = c.RunInputs(); err != nil {
			utils.Logger.Fatalf("Agent run inputs plugin error %s", err)
			return -1
		}

		if err = c.RunFilters(); err != nil {
			utils.Logger.Fatalf("Agent run filter plugin error %s", err)
			return -1
		}

		if err = c.RunOutputs(); err != nil {
			utils.Logger.Fatalf("Agent run output plugin error %s", err)
			return -1
		}
	}
	utils.Logger.Info("Agent started.")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	if ppid := os.Getppid(); ppid == *pid {
		if pp, err := os.FindProcess(ppid); err == nil {
			pp.Signal(syscall.SIGUSR1)
		}
	}

	running = true
	for running {
		select {
		case <-signalChan:
			utils.Logger.Info("Agent is shutting down.")
			running = false
		}
	}

	for _, c := range configs {
		if err = c.StopInputs(); err != nil {
			utils.Logger.Fatalf("Agent stop inputs plugin error %s", err)
			return -1
		}

		if err = c.StopFilters(); err != nil {
			utils.Logger.Fatalf("Agent stop filter plugin error %s", err)
			return -1
		}

		if err = c.StopOutputs(); err != nil {
			utils.Logger.Fatalf("Agent stop output plugin error %s", err)
			return -1
		}
	}
	utils.Logger.Info("Agent graceful down.")
	return 0
}
