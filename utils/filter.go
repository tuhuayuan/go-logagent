package utils

import (
	"errors"

	"github.com/codegangsta/inject"
)

// FilterPlugin interface of all filter plugin.
type FilterPlugin interface {
	TypePlugin
	Process(LogEvent) LogEvent
}

// FilterPluginConfig base struct of all filter plugin.
type FilterPluginConfig struct {
	TypePluginConfig
}

// FilterHandler type filter handler.
type FilterHandler interface{}

// customer chan type for exit.
type filterExitSignal chan bool

// customer chan type for exit.
type filterExitChan chan bool

var (
	// manage all filter handler
	mapFilterHandler = map[string]FilterHandler{}
)

// RegistFilterHandler regist handler by name.
func RegistFilterHandler(name string, handler FilterHandler) {
	mapFilterHandler[name] = handler
}

// RunFilters start filter.
func (c *Config) RunFilters() (err error) {
	c.Injector.Map(make(filterExitSignal, 1))
	c.Injector.Map(make(filterExitChan, 1))
	rvs, err := c.Injector.Invoke(c.runFilters)
	if !rvs[0].IsNil() {
		err = rvs[0].Interface().(error)
	}
	return
}

// StopFilters try to stop filter gracefully.
func (c *Config) StopFilters() (err error) {
	_, err = c.Injector.Invoke(c.stopFilters)
	return
}

func (c *Config) stopFilters(es filterExitSignal, ec filterExitChan) error {
	es <- true
	<-ec
	return nil
}

// runFilters.
func (c *Config) runFilters(inchan InChan, outchan OutChan, es filterExitSignal, ec filterExitChan) (err error) {
	filters, err := c.getFilters()
	if err != nil {
		Logger.Errorf("Run filters error %q", err)
		return
	}

	go func() {
		running := true
		for running {
			select {
			case event := <-inchan:
				for _, filter := range filters {
					// filter process on config order
					event = filter.Process(event)
				}
				outchan <- event
			case <-es:
				if len(inchan) == 0 {
					running = false
				}
				es <- true
			}
		}
		ec <- true
	}()
	return
}

// getFilters.
func (c *Config) getFilters() (filters []FilterPlugin, err error) {
	for _, part := range c.FilterPart {
		handler, ok := mapFilterHandler[part["type"].(string)]
		if !ok {
			return []FilterPlugin{}, errors.New("unknow filter type " + part["type"].(string))
		}

		inj := inject.New()
		inj.SetParent(c)
		inj.Map(&part)

		refvs, err := inj.Invoke(handler)
		if err != nil {
			return []FilterPlugin{}, err
		}

		for _, v := range refvs {
			if !v.CanInterface() || v.IsNil() {
				continue
			}
			if conf, ok := v.Interface().(FilterPlugin); ok {
				conf.SetInjector(inj)
				filters = append(filters, conf)
			}
		}
	}
	return
}
