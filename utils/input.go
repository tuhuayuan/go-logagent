package utils

import (
	"errors"

	"github.com/codegangsta/inject"
)

// InputPlugin base interface of input plugin.
type InputPlugin interface {
	TypePlugin
	// Start main entry for input plugin.
	Start()
	// Stop try stop plugin gracefully.
	Stop()
}

// InputPluginConfig base struct of input plugin config.
type InputPluginConfig struct {
	TypePluginConfig
}

// InputHandler type interface of input plugin implement.
// Injector map args *ConfigPart, InChan
type InputHandler interface{}

var (
	// hold all regist handler type
	mapInputHandler = map[string]InputHandler{}
)

// RegistInputHandler regist a input plugin implement.
func RegistInputHandler(name string, handler InputHandler) {
	mapInputHandler[name] = handler
}

// RunInputs run all inputs plugin.
func (c *Config) RunInputs() (err error) {
	rvs, err := c.Injector.Invoke(c.runInputs)
	if !rvs[0].IsNil() {
		err = rvs[0].Interface().(error)
	}
	return
}

// StopInputs try stop all running input plugin, block until gracefully stopped.
func (c *Config) StopInputs() (err error) {
	_, err = c.Injector.Invoke(c.stopInputs)
	return
}

// stopInputs
func (c *Config) stopInputs(inputs []InputPlugin) (err error) {
	for _, input := range inputs {
		input.Stop()
	}
	return
}

// runInputs.
func (c *Config) runInputs(inchan InChan) (err error) {
	// inchan is maped.
	inputs, err := c.getInputs(inchan)
	if err != nil {
		return
	}
	// prepare for stoping.
	c.Injector.Map(inputs)
	for _, input := range inputs {
		go input.Start()
	}
	return
}

// getInputs.
func (c *Config) getInputs(inchan InChan) (inputs []InputPlugin, err error) {
	for _, part := range c.InputPart {
		handler, ok := mapInputHandler[part["type"].(string)]
		if !ok {
			return []InputPlugin{}, errors.New("unknow input plugin type " + part["type"].(string))
		}

		// build input plugin injector.
		inj := inject.New()
		inj.SetParent(c)
		inj.Map(&part)
		inj.Map(inchan)

		// invoke plugin handler.
		refvs, err := inj.Invoke(handler)

		if err != nil {
			return []InputPlugin{}, err
		}

		for _, v := range refvs {
			if !v.CanInterface() || v.IsNil() {
				continue
			}

			if plugin, ok := v.Interface().(InputPlugin); ok {
				plugin.SetInjector(inj)
				inputs = append(inputs, plugin)
			}
		}
	}
	return
}
