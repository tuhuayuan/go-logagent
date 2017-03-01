package main

import (
	"flag"
	"strings"

	"zonst/tuhuayuan/logagent/utils"

	// load and regist all plugins
	"os"
	_ "zonst/tuhuayuan/logagent/filter/grok"
	_ "zonst/tuhuayuan/logagent/filter/patch"
	_ "zonst/tuhuayuan/logagent/input/file"
	_ "zonst/tuhuayuan/logagent/input/stdin"
	_ "zonst/tuhuayuan/logagent/output/elastic"
	_ "zonst/tuhuayuan/logagent/output/redis"
	_ "zonst/tuhuayuan/logagent/output/stdout"
)

var (
	sentinel   = flag.Bool("sentinel", false, "Sentinel mode.")
	configFile = flag.String("configs", "/etc/logagent", "Directory of config files.")
	etcdHosts  = flag.String("endpoints", "", "Endpoints of etcd.")
	agentName  = flag.String("name", "$HOSTNAME", "Global agent name.")
	level      = flag.Int("v", 3, "Logger level 0(panic)~5(debug).")
	_default   = flag.Bool("default", false, "Run default config.")
	help       = flag.Bool("help", false, "Print this message.")
)

func main() {
	flag.Parse()

	utils.SetLoggerLevel(*level)

	if *help {
		flag.Usage()
		os.Exit(0)
	}
	if *agentName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			utils.Logger.Fatalf("Agent name not set, get hostname error %s", err)
			os.Exit(1)
		}
		utils.Logger.Warn("Agent name not set use hostname %s", hostname)
		*agentName = hostname
	}

	var code int
	if *sentinel {
		code = runSentinel()
	} else {
		code = runAgent()
	}
	os.Exit(code)
}

// getEtcdList get host list from arguments.
func getEtcdList() []string {
	endpoints := strings.Split(*etcdHosts, ";")
	for i, v := range endpoints {
		endpoints[i] = strings.TrimSpace(v)
	}
	return endpoints
}

// getEtcdPath get agent config path from arguments.
func getEtcdPath() string {
	return "/zonst.org/logagent/" + *agentName + "/"
}
