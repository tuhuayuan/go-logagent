package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"zonst/tuhuayuan/logagent/utils"

	// load plugins
	_ "zonst/tuhuayuan/logagent/filter/grok"
	_ "zonst/tuhuayuan/logagent/filter/patch"
	_ "zonst/tuhuayuan/logagent/input/file"
	_ "zonst/tuhuayuan/logagent/input/stdin"
	_ "zonst/tuhuayuan/logagent/output/elastic"
	_ "zonst/tuhuayuan/logagent/output/redis"
	_ "zonst/tuhuayuan/logagent/output/stdout"
)

var (
	configFile = flag.String("config", "", "The local config file path.")
	etcdHosts  = flag.String("endpoints", "", "The etcd config server list.")
	autoReload = flag.Bool("reload", false, "If auto reload when config changed.")
	agentName  = flag.String("name", "", "Default agent name. You must change it if using etcd for config server.")
	verbose    = flag.Bool("v", false, "Show verbose log.")
	std        = flag.Bool("std", false, "The shortcut to use default stdin to stdout config template.")
	help       = flag.Bool("help", false, "Print the usages.")
)

func main() {
	flag.Parse()

	if *agentName == "" {
		Logger.Exit("You must setup an agent name.")
	}
	utils.CmdAgentName(*agentName)
	if *help != false {
		flag.Usage()
		os.Exit(0)
	}

	var confs []utils.Config
	if *std != false {
		conf, err := utils.LoadDefaultConfig()
		if err != nil {
			fmt.Println(err)
		}
		confs = append(confs, conf)
	} else {
		flag.Usage()
	}
	utils.CmdRun(confs)

	// Relay SIGINT SIGTERM SIGKILL to chExit.
	chExit := make(chan os.Signal)
	signal.Notify(chExit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-chExit:
		fmt.Println("Bye.")
	}
}
