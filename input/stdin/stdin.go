package stdininput

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/Sirupsen/logrus"

	"io"
	"zonst/tuhuayuan/logagent/utils"
)

const (
	// PluginName name of this plugin
	PluginName = "stdin"
)

type signalChan chan int

// PluginConfig Plugin Config struct of this plugin
type PluginConfig struct {
	utils.InputPluginConfig
	Prefix   string `json:"prefix"`
	hostname string
	running  bool
	exitChan signalChan
}

func init() {
	utils.RegistInputHandler(PluginName, InitHandler)
}

// InitHandler build handler.
func InitHandler(part *utils.ConfigPart) (plugin *PluginConfig, err error) {
	me := PluginConfig{
		InputPluginConfig: utils.InputPluginConfig{
			TypePluginConfig: utils.TypePluginConfig{
				Type: PluginName,
			},
		},
		exitChan: make(signalChan, 1),
	}
	if err = utils.ReflectConfigPart(part, &me); err != nil {
		return
	}

	if me.hostname, err = os.Hostname(); err != nil {
		return
	}

	plugin = &me
	return
}

// Start start it.
func (plugin *PluginConfig) Start() {
	plugin.Invoke(plugin.echo)
}

// Stop stop it.
func (plugin *PluginConfig) Stop() {
	plugin.running = false
	<-plugin.exitChan
}

func (plugin *PluginConfig) echo(logger *logrus.Logger, inchan utils.InChan) (err error) {
	defer func() {
		if err != nil {
			logger.Errorln(err)
		}
	}()

	plugin.running = true
	var reader *bufio.Reader
	if os.Getenv("TEST") != "" {
		tmpFile := os.Getenv("TEMPFILE")
		f, err := os.Open(tmpFile)
		if err != nil {
			utils.Logger.Warnf("StdinPlugin can't open temp file %s, fallback to stdin", tmpFile)
			reader = bufio.NewReader(os.Stdin)
		}
		reader = bufio.NewReader(f)
	} else {
		reader = bufio.NewReader(os.Stdin)
	}

	for plugin.running {
		fmt.Print(plugin.Prefix)
		data, _, err := reader.ReadLine()
		if err == io.EOF {
			plugin.running = false
			continue
		}
		command := string(data)
		event := utils.LogEvent{
			Timestamp: time.Now(),
			Message:   command,
			Extra: map[string]interface{}{
				"host": plugin.hostname,
			},
		}
		inchan <- event

	}
	plugin.exitChan <- 1
	return
}
