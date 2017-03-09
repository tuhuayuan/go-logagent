package stdininput

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"zonst/qipai/logagent/utils"
)

const (
	// PluginName name of this plugin
	PluginName = "stdin"
)

// PluginConfig Plugin Config struct of this plugin
type PluginConfig struct {
	utils.InputPluginConfig
	Prefix string `json:"prefix"`

	hostname  string
	exitChan  chan int
	inputChan chan string
}

func init() {
	utils.RegistInputHandler(PluginName, InitHandler)
}

// InitHandler create handler.
func InitHandler(part *utils.ConfigPart) (plugin *PluginConfig, err error) {
	config := PluginConfig{
		InputPluginConfig: utils.InputPluginConfig{
			TypePluginConfig: utils.TypePluginConfig{
				Type: PluginName,
			},
		},
		exitChan:  make(chan int),
		inputChan: make(chan string),
	}
	if err = utils.ReflectConfigPart(part, &config); err != nil {
		return
	}
	if config.hostname, err = os.Hostname(); err != nil {
		return
	}
	plugin = &config
	return
}

// Start start it.
func (plugin *PluginConfig) Start() {
	plugin.Invoke(plugin.loopRead)
}

// Stop stop it.
func (plugin *PluginConfig) Stop() {
	close(plugin.exitChan)
}

func (plugin *PluginConfig) loopRead(inChan utils.InputChannel) (err error) {
	go func(plugin *PluginConfig) {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Println(plugin.Prefix)
			data, _, err := reader.ReadLine()
			if err == io.EOF {
				return
			}
			plugin.inputChan <- string(data)
		}
	}(plugin)

	for {
		select {
		case <-plugin.exitChan:
			return
		case input := <-plugin.inputChan:
			event := utils.LogEvent{
				Timestamp: time.Now(),
				Message:   input,
				Extra: map[string]interface{}{
					"host": plugin.hostname,
				},
			}
			inChan.Input(event)
		}
	}
}
