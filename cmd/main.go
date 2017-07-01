package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	_ "github.com/tuhuayuan/go-logagent/filter/grok"
	_ "github.com/tuhuayuan/go-logagent/filter/patch"
	_ "github.com/tuhuayuan/go-logagent/filter/timezone"
	_ "github.com/tuhuayuan/go-logagent/input/file"
	_ "github.com/tuhuayuan/go-logagent/input/http"
	_ "github.com/tuhuayuan/go-logagent/input/stdin"
	_ "github.com/tuhuayuan/go-logagent/input/udp"
	_ "github.com/tuhuayuan/go-logagent/output/elastic"
	_ "github.com/tuhuayuan/go-logagent/output/redis"
	_ "github.com/tuhuayuan/go-logagent/output/stdout"

	"github.com/tuhuayuan/go-logagent/utils"
)

var (
	sentinel  = flag.Bool("sentinel", false, "Sentinel mode.")
	configDir = flag.String("configs", "/etc/logagent", "Directory of config files.")
	dataDir   = flag.String("data", "/var/logagent", "Directory of disk data to be store.")
	etcdHosts = flag.String("endpoints", "", "Endpoints of etcd.")
	agentName = flag.String("name", "", "Global agent name.")
	level     = flag.Int("v", 3, "Logger level 0(panic)~5(debug).")
	help      = flag.Bool("help", false, "Print this message.")
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		err       error
		hostname  string
		ag        *utils.Agent
		agStl     utils.Sentinel
		readyChan = make(chan int)
	)

	flag.Parse()

	utils.SetLoggerLevel(*level)
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	if *agentName == "" {
		hostname, err = os.Hostname()
		if err != nil {
			utils.Logger.Fatalf("Agent name not set, get hostname error %s", err)
			os.Exit(1)
		}
		utils.Logger.Warnf("Agent name not set use hostname %s", hostname)
		*agentName = hostname
	}
	// trace system signal.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		<-readyChan
		<-signalChan
		if *sentinel {
			agStl.Stop()
		} else {
			ag.Stop()
		}
	}()
	// create agent.
	ag = utils.NewAgent()
	ag.ConfigDir = *configDir
	ag.DataDir = *dataDir
	ag.EtcdHosts = *etcdHosts
	ag.Name = *agentName

	if *sentinel {
		agStl, err = ag.CreateSentinel()
		if err == nil {
			close(readyChan)
			err = agStl.Run()
		}
	} else {
		close(readyChan)
		err = ag.Run()
	}

	if err != nil {
		os.Exit(-1)
	} else {
		os.Exit(0)
	}
}
