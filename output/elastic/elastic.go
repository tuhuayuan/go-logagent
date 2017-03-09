package outputelastic

import (
	"zonst/qipai/logagent/utils"

	elastigo "github.com/mattbaird/elastigo/lib"
)

const (
	// PluginName name of this plugin
	PluginName = "elastic"
)

// PluginConfig plugin struct
type PluginConfig struct {
	utils.OutputPluginConfig

	Hosts    []string `json:"hosts"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Index    string   `json:"index"`
	DocType  string   `json:"doc_type"`

	conn         *elastigo.Conn
	bufChan      chan utils.LogEvent
	exitChan     chan int
	exitSyncChan chan int
}

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

// InitHandler create plugin.
func InitHandler(part *utils.ConfigPart) (plugin *PluginConfig, err error) {
	config := PluginConfig{
		OutputPluginConfig: utils.OutputPluginConfig{
			TypePluginConfig: utils.TypePluginConfig{
				Type: PluginName,
			},
		},
		conn:         elastigo.NewConn(),
		bufChan:      make(chan utils.LogEvent),
		exitChan:     make(chan int),
		exitSyncChan: make(chan int),
	}
	// read config
	err = utils.ReflectConfigPart(part, &config)
	if err != nil {
		utils.Logger.Errorf("Elastic plugin config error %q", err)
		return
	}
	// setup elastic client
	config.conn.SetHosts(config.Hosts)
	if config.Username != "" {
		config.conn.Username = config.Username
		config.conn.Password = config.Password
	}
	// test connection
	_, err = config.conn.Health("_all")
	if err != nil {
		utils.Logger.Warnf("Elasic cluster health check error %q", err)
	}
	plugin = &config
	return
}

// Process send log event.
func (plugin *PluginConfig) Process(ev utils.LogEvent) (err error) {
	var (
		index   string
		docType string
	)

	index = ev.Format(plugin.Index)
	docType = ev.Format(plugin.DocType)
	_, err = plugin.conn.Index(index, docType, "", map[string]interface{}{}, ev.GetMap())
	if err != nil {
		utils.Logger.Warnf("Elastic: output index error %q", err)
	}
	return
}

// Stop stop loop.
func (plugin *PluginConfig) Stop() {
	plugin.conn.Close()
}
