package utils

import (
	"bytes"
	"encoding/gob"
	"errors"
	"reflect"
	"sync"
	"time"

	"zonst/tuhuayuan/logagent/queue"

	"github.com/codegangsta/inject"
)

// OutputPlugin interface.
type OutputPlugin interface {
	TypePlugin
	Process(event LogEvent) error
	Stop()
}

type diskOutput struct {
	queue    queue.Queue
	exitChan chan int
	group    *sync.WaitGroup
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
	var (
		rets []reflect.Value
	)

	rets, err = c.Invoke(func(plugins []OutputPlugin, outputs map[string]*diskOutput) (err error) {
		for _, plugin := range plugins {
			dq := outputs[plugin.GetType()]
			buff := &bytes.Buffer{}
			gob.NewEncoder(buff).Encode(ev)
			if err != nil {
				return
			}
			// write diskqueue sync
			err = dq.queue.Put(buff.Bytes())
		}
		return
	})
	if err != nil {
		return
	}
	err = checkError(rets)
	return
}

// RunOutputs start output plugin.
func (c *Config) RunOutputs() (err error) {
	var queues = map[string]*diskOutput{}

	outputs, err := c.getOutputs()
	if err != nil {
		return
	}

	group := &sync.WaitGroup{}
	c.Map(queues)
	c.Map(group)
	for _, plugin := range outputs {
		dq := &diskOutput{
			exitChan: make(chan int),
			group:    group,
		}

		dq.queue = queue.New(c.Name+"_"+plugin.GetType(), c.DataPath,
			1024*1024*1024,
			0,
			1024*1024*10,
			1024,
			1*time.Second,
			Logger)
		queues[plugin.GetType()] = dq

		go func(dq *diskOutput, plugin OutputPlugin) {
			dq.group.Add(1)
			defer dq.group.Done()

			var (
				err     error
				running = true
			)

			for running {
				select {
				case raw := <-dq.queue.PeekChan():
					ev := LogEvent{}
					buff := &bytes.Buffer{}
					if _, err = buff.Write(raw); err != nil {
						goto next
					}
					if err = gob.NewDecoder(buff).Decode(&ev); err != nil {
						Logger.Warnf("Decoder return error %s", err)
						goto next
					}
					if err = plugin.Process(ev); err != nil {
						Logger.Warnf("Output process return error %s, retry in 5 sec.", err)
						time.Sleep(5 * time.Second)
						continue
					}
				next:
					<-dq.queue.ReadChan()
				case <-dq.exitChan:
					running = false
				}
			}
			dq.queue.Close()
		}(dq, plugin)
	}
	c.Map(outputs)
	return
}

// StopOutputs will block util gracefully stopped.
func (c *Config) StopOutputs() (err error) {
	_, err = c.Invoke(func(plugins []OutputPlugin, outputs map[string]*diskOutput, group *sync.WaitGroup) {
		for _, plugin := range plugins {
			plugin.Stop()
			dp := outputs[plugin.GetType()]
			dp.exitChan <- 1
		}
		group.Wait()
	})
	return
}

// getOutputs.
func (c *Config) getOutputs() (outputs []OutputPlugin, err error) {
	for _, part := range c.OutputPart {
		handler, ok := mapOutputHandler[part["type"].(string)]
		if !ok {
			err = errors.New("unknow output plugin type " + part["type"].(string))
			return
		}

		inj := inject.New()
		inj.SetParent(c)
		inj.Map(&part)

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
