package stdininput

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"zonst/tuhuayuan/logagent/utils"
)

const (
	// PluginName name of this plugin
	PluginName = "stdin"
)

// PluginConfig Plugin Config struct of this plugin
type PluginConfig struct {
	utils.InputPluginConfig
	Prefix string `json:"prefix"`

	hostname    string
	exitSignal  chan bool
	inputSignal chan string
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
		exitSignal:  make(chan bool, 1),
		inputSignal: make(chan string, 1),
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
	plugin.exitSignal <- true
}

func (plugin *PluginConfig) loopRead(inchan utils.InChan) (err error) {
	go func(plugin *PluginConfig) {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Println(plugin.Prefix)
			data, _, err := reader.ReadLine()
			if err == io.EOF {
				return
			}
			plugin.inputSignal <- string(data)
		}
	}(plugin)

	for {
		select {
		case <-plugin.exitSignal:
			return
		case input := <-plugin.inputSignal:
			event := utils.LogEvent{
				Timestamp: time.Now(),
				Message:   input,
				Extra: map[string]interface{}{
					"host": plugin.hostname,
				},
			}
			inchan <- event
		}
	}
}
