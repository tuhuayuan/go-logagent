package utils

import (
	"bytes"
	"encoding/gob"
	"errors"
	"time"

	"github.com/codegangsta/inject"

	"zonst/tuhuayuan/logagent/queue"
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

// Input implement InputChannel interface
func (c *Config) Input(ev LogEvent) (err error) {
	_, err = c.Invoke(func(dq queue.Queue, enc *gob.Encoder, buff *bytes.Buffer) (err error) {
		buff.Reset()
		err = enc.Encode(ev)
		if err != nil {
			return
		}
		// write diskqueue sync
		err = dq.Put(buff.Bytes())
		return
	})
	return
}

// RunInputs run all input plugin.
func (c *Config) RunInputs() (err error) {
	_, err = c.Invoke(c.runInputs)
	return
}

// StopInputs try stop all running input plugin, block until gracefully stopped.
func (c *Config) StopInputs() (err error) {
	_, err = c.Invoke(func(plugins []InputPlugin, dq queue.Queue) {
		for _, p := range plugins {
			p.Stop()
		}
		dq.Close()
	})
	return
}

// runInputs.
func (c *Config) runInputs() (err error) {
	inputs, err := c.getInputs(c)
	if err != nil {
		return
	}
	// setup coder
	var coderBuf bytes.Buffer
	enc := gob.NewEncoder(&coderBuf)
	dec := gob.NewDecoder(&coderBuf)
	// setup diskqueue
	dq := queue.New(c.Name, c.DataPath,
		1024*1024*1024,
		0,
		1024*1024*10,
		1024,
		1*time.Second,
		Logger)
	// inject input plugins and diskqueue
	c.Map(inputs)
	c.Map(dq)
	c.Map(&coderBuf)
	c.Map(enc)
	c.Map(dec)
	// start all input plugin in individual goroutine
	for _, input := range inputs {
		go input.Start()
	}
	return
}

// getInputs create all configed input plugins.
func (c *Config) getInputs(inChan InputChannel) (inputs []InputPlugin, err error) {
	for _, part := range c.InputPart {
		handler, ok := mapInputHandler[part["type"].(string)]
		if !ok {
			return []InputPlugin{},
				errors.New("unknow input plugin type " + part["type"].(string))
		}

		// build input plugin injector.
		inj := inject.New()
		inj.SetParent(c)
		// inject config part and InputChannel interface
		inj.Map(&part)
		c.Map(inChan)

		// invoke plugin create factory.
		refvs, _ := inj.Invoke(handler)
		err = checkError(refvs)
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
