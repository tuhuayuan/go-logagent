package inputudp

import (
	"encoding/binary"
	"net"
	"os"
	"time"
	"zonst/qipai/logagent/utils"
)

const (
	// PluginName name of this plugin
	PluginName = "udp"
)

// PluginConfig Plugin Config struct of this plugin
type PluginConfig struct {
	utils.InputPluginConfig
	Host  string `json:"host"`
	Port  string `json:"port"`
	Magic uint16 `json:"magic"`

	hostname   string
	dataChan   chan string
	exitSignal chan bool
	exitNotify chan bool
}

func init() {
	utils.RegistInputHandler(PluginName, InitHandler)
}

// InitHandler create plugin
func InitHandler(part *utils.ConfigPart) (plugin *PluginConfig, err error) {
	config := PluginConfig{
		InputPluginConfig: utils.InputPluginConfig{
			TypePluginConfig: utils.TypePluginConfig{
				Type: PluginName,
			},
		},

		dataChan:   make(chan string, 1),
		exitSignal: make(chan bool, 1),
		exitNotify: make(chan bool, 1),
	}
	if err = utils.ReflectConfigPart(part, &config); err != nil {
		return
	}
	if config.hostname, err = os.Hostname(); err != nil {
		return
	}
	if config.Host == "" {
		config.Host = "0.0.0.0"
	}
	plugin = &config
	return
}

// Start start it.
func (plugin *PluginConfig) Start() {
	plugin.Invoke(plugin.listen)
}

// Stop stop it.
func (plugin *PluginConfig) Stop() {
	plugin.exitSignal <- true
	<-plugin.exitNotify
}

// listen read data from udp emit logevent.
func (plugin *PluginConfig) listen(inChan utils.InputChannel) (err error) {
	addr, err := net.ResolveUDPAddr("udp", plugin.Host+":"+plugin.Port)

	if err != nil {
		utils.Logger.Errorf("Udp plugin addr error %s", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		utils.Logger.Errorf("Udp listen addr error %s", err)
		return
	}

	defer conn.Close()

	for {
		go plugin.handlerData(conn)

		select {
		case data := <-plugin.dataChan:
			inChan.Input(utils.LogEvent{
				Timestamp: time.Now(),
				Message:   data,
				Extra: map[string]interface{}{
					"host": plugin.hostname,
				},
			})

		case <-plugin.exitSignal:
			plugin.exitNotify <- true
			return
		}
	}
}

// handler udp data
func (plugin *PluginConfig) handlerData(conn *net.UDPConn) {
	data := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(data)
	if err != nil {
		utils.Logger.Warnf("Read data from udp error %s", err)
		return
	}
	if n > 2 && plugin.Magic == binary.BigEndian.Uint16(data[0:2]) {
		plugin.dataChan <- string(data[2:n])
	}
}
