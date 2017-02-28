package outputelastic

import (
	"zonst/tuhuayuan/logagent/utils"
)

const (
	// PluginName name of this plugin
	PluginName = "elastic"
)

// PluginConfig plugin struct
type PluginConfig struct {
	utils.OutputPluginConfig
}

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

// InitHandler create plugin.
func InitHandler() (plugin *PluginConfig, err error) {
	return
}

// Process send log event.
func (plugin *PluginConfig) Process(event utils.LogEvent) {

}

// Stop stop loop.
func (plugin *PluginConfig) Stop() {

}
