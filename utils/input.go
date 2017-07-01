package utils

import (
	"errors"
	"reflect"

	"github.com/codegangsta/inject"
)

// InputPlugin interface.
type InputPlugin interface {
	TypePlugin
	Start()
	Stop()
}

// InputPluginConfig base struct of input plugin config.
type InputPluginConfig struct {
	TypePluginConfig
}

// InputHandler factory interface type
type InputHandler interface{}

var (
	mapInputHandler = map[string]InputHandler{}
)

// RegistInputHandler regist a input plugin type factory.
func RegistInputHandler(name string, handler InputHandler) {
	mapInputHandler[name] = handler
}

// RunInputs run all input plugin.
func (c *Config) RunInputs() (err error) {
	var (
		rets []reflect.Value
	)
	rets, err = c.Invoke(c.runInputs)
	if err != nil {
		return
	}
	err = CheckError(rets)
	return
}

// StopInputs try stop all running input plugin, block until gracefully stopped.
func (c *Config) StopInputs() (err error) {
	var (
		rets []reflect.Value
	)
	rets, err = c.Invoke(func(plugins []InputPlugin) {
		for _, p := range plugins {
			p.Stop()
		}
	})
	if err != nil {
		return
	}
	err = CheckError(rets)
	return
}

// runInputs.
func (c *Config) runInputs() (err error) {
	inputs, err := c.getInputs()
	if err != nil {
		return
	}
	c.Map(inputs)
	// start all input plugin in individual goroutine
	for _, input := range inputs {
		go input.Start()
	}
	return
}

// getInputs create all configed input plugins.
func (c *Config) getInputs() (inputs []InputPlugin, err error) {
	var (
		rets []reflect.Value
	)

	for _, part := range c.InputPart {
		handler, ok := mapInputHandler[part["type"].(string)]
		if !ok {
			return []InputPlugin{},
				errors.New("unknow input plugin type " + part["type"].(string))
		}

		// build input plugin injector.
		inj := inject.New()
		inj.SetParent(c)
		inj.Map(&part)

		// invoke plugin create factory.
		if rets, err = inj.Invoke(handler); err != nil {
			return
		}
		if err = CheckError(rets); err != nil {
			return
		}

		for _, v := range rets {
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
