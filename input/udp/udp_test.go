package inputudp

import (
	"encoding/binary"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"zonst/tuhuayuan/logagent/queue"
	"zonst/tuhuayuan/logagent/utils"
)

func init() {
	utils.RegistInputHandler(PluginName, InitHandler)
}

func Test_Udp(t *testing.T) {
	plugin, err := utils.LoadFromString(`{
		"input": [{
			"type": "udp",
			"host": "0.0.0.0",
            "port": "10020",
            "magic": 16

		}]
	}`)
	assert.NoError(t, err)
	err = plugin.RunInputs()
	assert.NoError(t, err)

	raddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:10020")
	conn, err := net.DialUDP("udp", nil, raddr)
	assert.NoError(t, err)
	defer conn.Close()
	data := []byte("  Log message 消息.")
	binary.BigEndian.PutUint16(data, 16)
	n, err := conn.Write(data)
	assert.Equal(t, 21, n)
	plugin.Invoke(func(dq queue.Queue) {
		fmt.Println(<-dq.ReadChan())
	})

}
