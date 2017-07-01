package filtertimezone

// change timezone for event's @timestamp

import (
	"time"

	"github.com/tuhuayuan/go-logagent/utils"
)

const (
	// PluginName name of this patch
	PluginName = "timezone"
)

// PluginConfig struct of plugin config
type PluginConfig struct {
	utils.FilterPluginConfig

	Timezone string `json:"timezone"`

	offset int
}

func init() {
	utils.RegistFilterHandler(PluginName, InitHandler)
}

// InitHandler create plugin.
func InitHandler(part *utils.ConfigPart) (plugin *PluginConfig, err error) {
	config := PluginConfig{
		FilterPluginConfig: utils.FilterPluginConfig{
			TypePluginConfig: utils.TypePluginConfig{
				Type: PluginName,
			},
		},
	}
	// Reflect config from configraw.
	if err = utils.ReflectConfigPart(part, &config); err != nil {
		return
	}
	if config.Timezone == "" {
		config.Timezone = "UTC"
	}
	locat, err := time.LoadLocation(config.Timezone)
	if err != nil {
		return
	}
	t1, err := time.ParseInLocation("2006-01-02", "2017-03-05", locat)
	if err != nil {
		return
	}
	_, config.offset = t1.Zone()

	plugin = &config
	return
}

// Process do pathing
func (plugin *PluginConfig) Process(event utils.LogEvent) utils.LogEvent {
	if plugin.offset != 0 {
		event.Timestamp = event.Timestamp.Add(time.Duration(plugin.offset) * time.Second)
	}

	return event
}
