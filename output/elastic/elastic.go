package outputelastic

import (
	"context"

	"zonst/qipai/logagent/utils"

	elastic "gopkg.in/olivere/elastic.v5"
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

	conn         *elastic.Client
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

	config.conn, err = elastic.NewClient(
		elastic.SetURL(config.Hosts...),
		elastic.SetBasicAuth(config.Username, config.Password),
	)
	if err != nil {
		utils.Logger.Warnf("Elasic cluster health check error %q", err)
	}
	plugin = &config
	return
}

// Process send log event.
func (plugin *PluginConfig) Process(ev utils.LogEvent) (err error) {
	var (
		docIndex string
		docType  string
		docID    string
	)

	docIndex = ev.Format(ev.Extra["@elastic_docindex"].(string))
	docType = ev.Format(ev.Extra["@elastic_doctype"].(string))
	docID = ev.Format(ev.Extra["@elastic_docid"].(string))
	_, err = plugin.conn.Index().
		Index(docIndex).
		Type(docType).
		Id(docID).
		BodyString(ev.Message).
		Do(context.Background())
	if err != nil {
		utils.Logger.Warnf("Elastic: output index error %q", err)
	}
	return
}

// Stop stop loop.
func (plugin *PluginConfig) Stop() {
	if plugin.conn != nil {
		plugin.conn.Stop()
	}
}
