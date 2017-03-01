package utils

import (
	"errors"

	"github.com/codegangsta/inject"
)

// OutputPlugin base type interface of output plugin.
type OutputPlugin interface {
	TypePlugin
	Process(event LogEvent) (err error)
	Stop()
}

// OutputPluginConfig base type struct of output plugin config.
type OutputPluginConfig struct {
	TypePluginConfig
}

// OutputHandler type interface.
type OutputHandler interface{}

// exit signal
type outputExitSignal chan bool

// exit notify
type outputExitChan chan bool

var (
	// manage all output handler
	mapOutputHandler = map[string]OutputHandler{}
)

// RegistOutputHandler regist handler by name.
func RegistOutputHandler(name string, handler OutputHandler) {
	mapOutputHandler[name] = handler
}

// RunOutputs start output plugin.
func (c *Config) RunOutputs() (err error) {
	c.Injector.Map(make(outputExitChan, 1))
	c.Injector.Map(make(outputExitSignal, 1))
	_, err = c.Injector.Invoke(c.runOutputs)
	return
}

// StopOutputs stop all plugin this will block util gracefully stopped.
func (c *Config) StopOutputs() (err error) {
	// finish all message in InChan
	_, err = c.Injector.Invoke(func(exitSignal outputExitSignal, exitNotify outputExitChan) {
		exitSignal <- true
		<-exitNotify
	})
	// stop all plugin
	_, err = c.Injector.Invoke(func(outputs []OutputPlugin) {
		for _, plugin := range outputs {
			plugin.Stop()
		}
	})
	return
}

// runOutputs.
func (c *Config) runOutputs(outchan OutChan, exitSignal outputExitSignal, exitNotify outputExitChan) (err error) {
	outputs, err := c.getOutputs()
	if err != nil {
		return
	}
	// prepare for stoping.
	c.Injector.Map(outputs)
	running := true
	go func() {
		for running {
			select {
			case event := <-outchan:
				for _, output := range outputs {
					if err = output.Process(event); err != nil {
						Logger.Errorf("output plugin failed: %q\n", err)
					}
				}
			case <-exitSignal:
				if len(outchan) == 0 {
					running = false
				}
				exitSignal <- true
			}
		}
		exitNotify <- true
	}()
	return
}

// getOutputs.
func (c *Config) getOutputs() (outputs []OutputPlugin, err error) {
	for _, part := range c.OutputPart {
		handler, ok := mapOutputHandler[part["type"].(string)]
		if !ok {
			err = errors.New(part["type"].(string))
			return
		}

		inj := inject.New()
		inj.SetParent(c)
		inj.Map(&part)

		refvs, err := inj.Invoke(handler)
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
