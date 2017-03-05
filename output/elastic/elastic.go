package outputelastic

import (
	"zonst/tuhuayuan/logagent/utils"

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
	// TODO if elastic index not created, using template settings create it.
	Template string `json:"template"`
	Index    string `json:"index"`
	DocType  string `json:"doc_type"`

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
	conf := PluginConfig{
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
	err = utils.ReflectConfigPart(part, &conf)
	if err != nil {
		utils.Logger.Errorf("Elastic plugin config error %q", err)
		return
	}
	// setup elastic client
	conf.conn.SetHosts(conf.Hosts)
	if conf.Username != "" {
		conf.conn.Username = conf.Username
		conf.conn.Password = conf.Password
	}
	// test connection
	_, err = conf.conn.Health("_all")
	if err != nil {
		utils.Logger.Warnf("Elasic cluster health check error %q", err)
	}
	plugin = &conf
	go plugin.loopEvent()
	return
}

// Process send log event.
func (plugin *PluginConfig) Process(event utils.LogEvent) (err error) {
	plugin.bufChan <- event
	return
}

// Stop stop loop.
func (plugin *PluginConfig) Stop() {
	plugin.exitChan <- 1
	<-plugin.exitSyncChan
}

// loopEvent
func (plugin *PluginConfig) loopEvent() (err error) {
	var (
		index   string
		docType string
	)

	for {
		select {
		case event := <-plugin.bufChan:
			index = event.Format(plugin.Index)
			docType = event.Format(plugin.DocType)
			_, err := plugin.conn.Index(index, docType, "", map[string]interface{}{}, event.GetMap())
			// TODO elastic error not handler.
			if err != nil {
				utils.Logger.Warnf("Elastic: output index error %q", err)
			}

		case <-plugin.exitChan:
			plugin.conn.Close()
			close(plugin.bufChan)
			close(plugin.exitSyncChan)
			return
		}
	}
}
