package utils

import (
	"bytes"
	"encoding/gob"
	"errors"

	"github.com/codegangsta/inject"

	"time"
	"zonst/tuhuayuan/logagent/queue"
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

// RunFilters run all filter plugin.
func (c *Config) RunFilters() (err error) {
	c.Map(make(filterExitChan))
	c.Map(make(filterExitSyncChan))
	_, err = c.Invoke(c.runFilters)
	return
}

// StopFilters try to stop filter gracefully.
func (c *Config) StopFilters() (err error) {
	_, err = c.Invoke(func(exit filterExitChan, exitSync filterExitSyncChan) {
		exit <- 1
		<-exitSync
	})
	return
}

// runFilters.
func (c *Config) runFilters(outChan OutputChannel,
	dq queue.Queue, buf *bytes.Buffer, dec *gob.Decoder,
	exit filterExitChan, exitSync filterExitSyncChan) (err error) {
	filters, err := c.getFilters()
	if err != nil {
		return
	}
	running := true
	tick := time.NewTicker(100 * time.Millisecond)
	go func() {
		for running {
			select {
			case raw := <-dq.PeekChan():
				event := LogEvent{}
				buf.Reset()
				if _, err = buf.Write(raw); err != nil {
					goto next
				}
				if err = dec.Decode(&event); err != nil {
					goto next
				}
				for _, filter := range filters {
					event = filter.Process(event)
				}
				if err = outChan.Output(event); err != nil {
					Logger.Warn("Filter output return error %s, message retry.", err)
					continue
				}
			next:
				<-dq.ReadChan()
			case <-tick.C:
				// tick
			case <-exit:
				running = false
			}
		}
		close(exitSync)
	}()
	return
}

// getFilters.
func (c *Config) getFilters() (filters []FilterPlugin, err error) {
	for _, part := range c.FilterPart {
		handler, ok := mapFilterHandler[part["type"].(string)]
		if !ok {
			return []FilterPlugin{},
				errors.New("unknow filter type " + part["type"].(string))
		}

		inj := inject.New()
		inj.SetParent(c)
		inj.Map(&part)

		refvs, _ := inj.Invoke(handler)
		err = checkError(refvs)
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
