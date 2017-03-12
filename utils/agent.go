package utils

// Agent 结构体.
type Agent struct {
	Name      string
	ConfigDir string
	DataDir   string
	EtcdHosts string

	configs      []Config
	exitChan     chan int
	exitSyncChan chan int
}

// NewAgent 创建agent
func NewAgent() *Agent {
	ag := &Agent{
		exitChan:     make(chan int),
		exitSyncChan: make(chan int),
	}
	return ag
}

// Run 运行Agent
func (ag *Agent) Run() (err error) {
	if ag.EtcdHosts != "" {
		ag.configs, err = LoadFromNode(getEtcdList(ag.EtcdHosts), getEtcdPath(ag.Name), ag.DataDir)
	} else {
		ag.configs, err = LoadFromDir(ag.ConfigDir, ag.DataDir)
	}

	if err != nil {
		Logger.Fatalf("Agent load config error %s", err)
		return
	}

	// 启动主要组件
	for _, c := range ag.configs {
		if err = c.RunInputs(); err != nil {
			Logger.Fatalf("Agent run inputs plugin error %s", err)
			return
		}
		if err = c.RunOutputs(); err != nil {
			Logger.Fatalf("Agent run output plugin error %s", err)
			return
		}
		if err = c.RunFilters(); err != nil {
			Logger.Fatalf("Agent run filter plugin error %s", err)
			return
		}
	}
	Logger.Info("Agent started.")
	<-ag.exitChan
	Logger.Info("Agent is shutting down.")

	for _, c := range ag.configs {
		if err = c.StopInputs(); err != nil {
			Logger.Fatalf("Agent stop inputs plugin error %s", err)
			return
		}

		if err = c.StopFilters(); err != nil {
			Logger.Fatalf("Agent stop filter plugin error %s", err)
			return
		}

		if err = c.StopOutputs(); err != nil {
			Logger.Fatalf("Agent stop output plugin error %s", err)
			return
		}
	}
	Logger.Info("Agent graceful down.")
	close(ag.exitSyncChan)
	return
}

// Stop 停止Agent
func (ag *Agent) Stop() {
	close(ag.exitChan)
	<-ag.exitSyncChan
}
