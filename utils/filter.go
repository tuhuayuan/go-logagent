package utils

import (
	"errors"
	"reflect"

	"github.com/codegangsta/inject"
)

// FilterPlugin interface.
type FilterPlugin interface {
	TypePlugin
	Process(LogEvent) LogEvent
}

// FilterPluginConfig base struct of all filter plugin.
type FilterPluginConfig struct {
	TypePluginConfig
}

// FilterHandler fctory interface type
type FilterHandler interface{}

// customer channel type for inject
type filterExitChan chan int
type filterExitSyncChan chan int

var (
	mapFilterHandler = map[string]FilterHandler{}
)

// RegistFilterHandler regist handler by name.
func RegistFilterHandler(name string, handler FilterHandler) {
	mapFilterHandler[name] = handler
}

// Input implement InputChannel interface
func (c *Config) Input(ev LogEvent) (err error) {
	var (
		rets []reflect.Value
	)
	// sync to OutputChannel
	rets, err = c.Invoke(func(outChan OutputChannel, filters []FilterPlugin) (err error) {
		for _, f := range filters {
			ev = f.Process(ev)
		}
		err = outChan.Output(ev)
		return
	})
	if err != nil {
		return
	}
	err = CheckError(rets)
	return
}

// RunFilters run all filter plugin.
func (c *Config) RunFilters() (err error) {
	var (
		rets []reflect.Value
	)
	rets, err = c.Invoke(func() (err error) {
		filters, err := c.getFilters()
		c.Map(filters)
		return
	})
	err = CheckError(rets)
	return
}

// StopFilters try to stop filter gracefully.
func (c *Config) StopFilters() (err error) {
	return
}

// getFilters.
func (c *Config) getFilters() (filters []FilterPlugin, err error) {
	var (
		rets []reflect.Value
	)

	for _, part := range c.FilterPart {
		handler, ok := mapFilterHandler[part["type"].(string)]
		if !ok {
			return []FilterPlugin{},
				errors.New("unknow filter type " + part["type"].(string))
		}

		inj := inject.New()
		inj.SetParent(c)
		inj.Map(&part)

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
			if plugin, ok := v.Interface().(FilterPlugin); ok {
				plugin.SetInjector(inj)
				filters = append(filters, plugin)
			}
		}
	}
	return
}
