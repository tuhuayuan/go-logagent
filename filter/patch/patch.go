package patchfilter

import (
	"zonst/qipai/logagent/utils"
)

// 暂时没发现有什么用，用来测试的
const (
	// PluginName name of this patch
	PluginName = "patch"
)

// PluginConfig struct of plugin config
type PluginConfig struct {
	utils.FilterPluginConfig
	Key   string `json:"key"`
	Value string `json:"value"`
}

func init() {
	utils.RegistFilterHandler(PluginName, InitHandler)
}

// InitHandler create plugin.
func InitHandler(part *utils.ConfigPart) (plugin *PluginConfig, err error) {
	conf := PluginConfig{
		FilterPluginConfig: utils.FilterPluginConfig{
			TypePluginConfig: utils.TypePluginConfig{
				Type: PluginName,
			},
		},
	}
	// Reflect config from configraw.
	if err = utils.ReflectConfigPart(part, &conf); err != nil {
		return
	}
	plugin = &conf
	return
}

// Process do pathing
func (plugin *PluginConfig) Process(event utils.LogEvent) utils.LogEvent {
	if _, ok := event.Extra[plugin.Key]; ok {
		return event
	}
	if event.Extra == nil {
		event.Extra = make(map[string]interface{})
	}
	event.Extra[plugin.Key] = event.Format(plugin.Value)
	return event
}
