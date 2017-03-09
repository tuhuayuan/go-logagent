package grokfilter

// TODO: I want be a grok clone, but later will be. :(
// https://github.com/logstash-plugins/logstash-patterns-core/tree/master/patterns

import (
	"regexp"

	"zonst/qipai/logagent/utils"
)

const (
	// PluginName name of this filter
	PluginName = "grok"
)

// PluginConfig filter config struct
type PluginConfig struct {
	utils.FilterPluginConfig
	Match     string `json:"match"`
	MaxFileds int    `json:"maxfields"`

	re *regexp.Regexp
}

func init() {
	utils.RegistFilterHandler(PluginName, InitHandler)
}

// InitHandler create plugin handler.
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
	// PreCompile
	if conf.re, err = regexp.Compile(conf.Match); err != nil {
		utils.Logger.Warnf("Grok filter regexp compile error: %s", err)
		return
	}

	if conf.MaxFileds <= 0 {
		conf.MaxFileds = 64
	}

	plugin = &conf
	return
}

// Process process logevent
func (plugin *PluginConfig) Process(event utils.LogEvent) utils.LogEvent {
	if event.Extra == nil {
		event.Extra = make(map[string]interface{})
	}
	maches := plugin.re.FindAllStringSubmatchIndex(event.Message, plugin.MaxFileds)
	names := plugin.re.SubexpNames()

	if maches != nil && len(maches[0])/2 >= len(names) {
		indexPairs := maches[0][2:]
		names = names[1:]
		for i, v := range names {
			i = i * 2
			event.Extra[v] = string(event.Message[indexPairs[i]:indexPairs[i+1]])
		}
	}

	return event
}
