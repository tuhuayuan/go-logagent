package outputredis

import (
	"time"

	"github.com/garyburd/redigo/redis"

	"zonst/qipai/logagent/utils"
)

const (
	// PluginName name of this plugin.
	PluginName = "redis"
)

// PluginConfig config struct.
type PluginConfig struct {
	utils.OutputPluginConfig
	Key      string `json:"key"`
	Host     string `json:"host"`
	DB       int    `json:"db"`
	Password string `json:"password"`
	DataType string `json:"data_type"`
	Timeout  int    `json:"timeout"`

	pool         *redis.Pool
	bufChan      chan utils.LogEvent
	exitChan     chan int
	exitSyncChan chan int
}

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

// InitHandler create plugin.
func InitHandler(part *utils.ConfigPart) (plugin *PluginConfig, err error) {
	conf := PluginConfig{
		OutputPluginConfig: utils.OutputPluginConfig{
			TypePluginConfig: utils.TypePluginConfig{
				Type: PluginName,
			},
		},

		bufChan:      make(chan utils.LogEvent),
		exitChan:     make(chan int),
		exitSyncChan: make(chan int),
	}
	if err = utils.ReflectConfigPart(part, &conf); err != nil {
		return
	}

	// init connection pool
	conf.pool = &redis.Pool{
		MaxIdle:     16,
		MaxActive:   16,
		IdleTimeout: 60 * time.Second,
		Dial: func() (redis.Conn, error) {

			ops := []redis.DialOption{
				redis.DialConnectTimeout(time.Second * 5),
				redis.DialDatabase(conf.DB),
			}
			if conf.Password != "" {
				ops = append(ops, redis.DialPassword(conf.Password))
			}
			conn, err := redis.Dial("tcp", conf.Host, ops...)
			if err != nil {
				utils.Logger.Warnf("Redis output dial redis error %q", err)
				return nil, err
			}
			return conn, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	plugin = &conf
	return
}

// Process flush log event
func (plugin *PluginConfig) Process(ev utils.LogEvent) (err error) {
	var (
		conn redis.Conn
		data []byte
		key  string
	)

	if data, err = ev.Marshal(true); err != nil {
		utils.Logger.Errorf("marshal failed: %v", ev)
		return
	}
	// get store key
	key = ev.Format(plugin.Key)
	// get a connection
	conn = plugin.pool.Get()
	if err = conn.Err(); err != nil {
		return
	}
	defer conn.Close()

	// types
	switch plugin.DataType {
	case "list":
		_, err = conn.Do("rpush", key, data)
	case "channel":
		_, err = conn.Do("publish", key, data)
	}
	if err != nil {
		utils.Logger.Warnf("Redis error %q, log lost.", err)
	}
	return
}

// Stop stop loopEvent goroutine
func (plugin *PluginConfig) Stop() {
	plugin.pool.Close()
}
