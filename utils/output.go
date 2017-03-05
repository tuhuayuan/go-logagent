package utils

import (
	"errors"

	"github.com/codegangsta/inject"
)

// OutputPlugin interface.
type OutputPlugin interface {
	TypePlugin
	Process(event LogEvent) error
	Stop()
}

// OutputPluginConfig base type struct of output plugin config.
type OutputPluginConfig struct {
	TypePluginConfig
}

// OutputHandler factory interface type
type OutputHandler interface{}

var (
	mapOutputHandler = map[string]OutputHandler{}
)

// RegistOutputHandler regist handler by name.
func RegistOutputHandler(name string, handler OutputHandler) {
	mapOutputHandler[name] = handler
}

// Output implement OutputChannel interface
func (c *Config) Output(ev LogEvent) (err error) {
	_, err = c.Invoke(func(outputs []OutputPlugin) (err error) {
		for _, plugin := range outputs {
			err = plugin.Process(ev)
			if err != nil {
				break
			}
		}
		return
	})
	return
}

// RunOutputs start output plugin.
func (c *Config) RunOutputs() (err error) {
	outputs, err := c.getOutputs(c)
	if err != nil {
		return
	}
	c.Map(outputs)
	return
}

// StopOutputs will block util gracefully stopped.
func (c *Config) StopOutputs() (err error) {
	_, err = c.Invoke(func(outputs []OutputPlugin) {
		for _, plugin := range outputs {
			plugin.Stop()
		}
	})
	return
}

// getOutputs.
func (c *Config) getOutputs(outChan OutputChannel) (outputs []OutputPlugin, err error) {
	for _, part := range c.OutputPart {
		handler, ok := mapOutputHandler[part["type"].(string)]
		if !ok {
			err = errors.New("unknow output plugin type " + part["type"].(string))
			return
		}

		inj := inject.New()
		inj.SetParent(c)
		inj.Map(&part)
		c.Map(outChan)

		refvs, _ := inj.Invoke(handler)
		checkError(refvs)
		if err != nil {
			return []OutputPlugin{}, err
		}

		for _, v := range refvs {
			if !v.CanInterface() || v.IsNil() {
				continue
			}
			if conf, ok := v.Interface().(OutputPlugin); ok {
				conf.SetInjector(inj)
				outputs = append(outputs, conf)
			}
		}
	}
	return
}
