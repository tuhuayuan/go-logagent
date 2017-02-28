package outputstdout

import (
	"fmt"

	"zonst/tuhuayuan/logagent/utils"
)

const (
	// PluginName name of this plugin
	PluginName = "stdout"
)

// PluginConfig struct of this plugin.
type PluginConfig struct {
	utils.OutputPluginConfig
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
	}

	if err = utils.ReflectConfigPart(part, &conf); err != nil {
		utils.Logger.Errorf("Patch output plugin create error %q", err)
		return
	}
	plugin = &conf
	return
}

// Process fluch event to stdout
func (plugin *PluginConfig) Process(event utils.LogEvent) (err error) {
	data, err := event.Marshal(true)
	if err != nil {
		return
	}
	fmt.Println(string(data))
	return
}

// Stop notthing need clean.
func (plugin *PluginConfig) Stop() {

}
