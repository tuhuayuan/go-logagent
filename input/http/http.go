package inputhttp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
	"zonst/tuhuayuan/logagent/utils"
)

const (
	// PluginName name of this plugin
	PluginName = "http"
)

// PluginConfig Plugin Config struct of this plugin
type PluginConfig struct {
	utils.InputPluginConfig
	Host    string   `json:"host"`
	URLPath string   `json:"url_path"`
	Methods []string `json:"methods"`

	hostname     string
	httpChan     chan utils.LogEvent
	exitChan     chan int
	exitSyncChan chan int
}

func init() {
	utils.RegistOutputHandler(PluginName, InitHandler)
}

// InitHandler create plugin
func InitHandler(part *utils.ConfigPart) (plugin *PluginConfig, err error) {
	config := PluginConfig{
		InputPluginConfig: utils.InputPluginConfig{
			TypePluginConfig: utils.TypePluginConfig{
				Type: PluginName,
			},
		},
		httpChan:     make(chan utils.LogEvent),
		exitChan:     make(chan int),
		exitSyncChan: make(chan int),
	}
	if err = utils.ReflectConfigPart(part, &config); err != nil {
		return
	}
	if config.hostname, err = os.Hostname(); err != nil {
		return
	}
	if config.URLPath == "" {
		config.URLPath = "/"
	}
	if config.Host == "" {
		config.Host = "0.0.0.0:3000"
	}
	if len(config.Methods) == 0 {
		config.Methods = []string{"PUT"}
	}
	for i, v := range config.Methods {
		config.Methods[i] = strings.ToUpper(v)
	}

	plugin = &config
	return
}

// Start start it.
func (plugin *PluginConfig) Start() {
	_, err := plugin.Invoke(func(inChan utils.InputChannel) {
		var mux = http.NewServeMux()
		mux.HandleFunc(plugin.URLPath, plugin.Handler)
		go func(serverAddr string, m *http.ServeMux) {
			utils.Logger.Infof("Http start listen at %s", serverAddr)
			if err := http.ListenAndServe(serverAddr, m); err != nil {
				utils.Logger.Warnf("Http start error %s", err)
			}
		}(plugin.Host, mux)

		for {
			select {
			case ev := <-plugin.httpChan:
				inChan.Input(ev)
			case <-plugin.exitChan:
				close(plugin.exitSyncChan)
				return
			}
		}
	})
	if err != nil {
		utils.Logger.Warnf("Http start error %s", err)
	}
}

// Stop stop it.
func (plugin *PluginConfig) Stop() {
	close(plugin.exitChan)
	<-plugin.exitSyncChan
}

// Handler http request handler
func (plugin *PluginConfig) Handler(w http.ResponseWriter, r *http.Request) {
	var (
		methodMatched bool
		err           error
	)
	for _, methodSupported := range plugin.Methods {
		if methodSupported == r.Method {
			methodMatched = true
			break
		}
	}
	if !methodMatched {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	typeAndCodec := strings.Split(contentType, ";")
	ev := utils.LogEvent{
		Timestamp: time.Now(),
		Extra: map[string]interface{}{
			"host":     plugin.hostname,
			"clientIP": r.RemoteAddr,
		},
	}
	switch typeAndCodec[0] {
	case "application/json":
		var raw []byte
		if raw, err = ioutil.ReadAll(r.Body); err == nil {
			objs := map[string]interface{}{}
			if err = json.Unmarshal(raw, &objs); err == nil {

				for k, v := range objs {
					fmt.Println(k, v)
					ev.Extra[k] = v
				}
			}
		}
	case "application/x-www-form-urlencoded":
		r.ParseForm()
		for k, v := range r.Form {
			ev.Extra[k] = v
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("[Content-Type:" + contentType + "] not accept."))
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	plugin.httpChan <- ev
	resp, err := json.MarshalIndent(ev.Extra, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(resp)
	return
}
