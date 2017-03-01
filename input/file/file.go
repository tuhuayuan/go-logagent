package fileinput

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"zonst/tuhuayuan/logagent/utils"
)

const (
	//PluginName name of this plugin
	PluginName = "file"
)

var (
	// manage fsnotify watcher
	mapWatcher = map[string]*fsnotify.Watcher{}
)

// SinceDBInfo struct of log file offset
type SinceDBInfo struct {
	Offset int64 `json:"offset"`
}

// PluginConfig config of this plugin.
type PluginConfig struct {
	utils.InputPluginConfig
	DirsPath  []string `json:"dirspath"`  // directory of log files
	FileType  string   `json:"filetype"`  // file suffix looking for
	Follow    bool     `json:"follow"`    // is follow new log or read from begining
	SincePath string   `json:"sincepath"` // since store path
	Intervals int      `json:"intervals"` // interval seconds of write sincdb

	hostname          string
	SinceDBInfos      map[string]*SinceDBInfo
	sinceLastInfos    []byte
	SinceLastSaveTime time.Time
	running           bool
	wgExit            *sync.WaitGroup
	dbInfosLock       *sync.RWMutex
}

func init() {
	utils.RegistInputHandler(PluginName, InitHandler)
}

// InitHandler create the plugin.
func InitHandler(part *utils.ConfigPart) (plugin *PluginConfig, err error) {
	me := PluginConfig{
		InputPluginConfig: utils.InputPluginConfig{
			TypePluginConfig: utils.TypePluginConfig{
				Type: PluginName,
			},
		},
		SinceDBInfos: map[string]*SinceDBInfo{},
		running:      true,
		wgExit:       &sync.WaitGroup{},
		dbInfosLock:  &sync.RWMutex{},
	}
	if err = utils.ReflectConfigPart(part, &me); err != nil {
		return
	}
	if me.Intervals == 0 {
		me.Intervals = 1
	}
	if me.FileType == "" {
		me.FileType = "log"
	}
	if me.hostname, err = os.Hostname(); err != nil {
		return
	}
	plugin = &me
	return
}

// Start start plugin.
func (plugin *PluginConfig) Start() {
	plugin.Invoke(plugin.watch)
}

// Stop stop plugin.
func (plugin *PluginConfig) Stop() {
	plugin.running = false
	for _, m := range mapWatcher {
		m.Close()
	}
	plugin.wgExit.Wait()
}

// load current sincedb data.
func (plugin *PluginConfig) loadSinceDB() (err error) {
	var (
		data []byte
	)
	plugin.SinceDBInfos = map[string]*SinceDBInfo{}

	if plugin.SincePath == "" || plugin.SincePath == "/dev/null" {
		utils.Logger.Warnf("Sincdb path miss config")
		return
	}

	if _, err = os.Stat(plugin.SincePath); err != nil {
		if os.IsNotExist(err) {
			// create file
			var f *os.File
			f, err = os.Create(plugin.SincePath)
			if err != nil {
				utils.Logger.Errorf("Create sincdb file error %s", err)
				return
			}
			f.WriteString("{}")
			f.Close()
		} else {
			utils.Logger.Errorf("Sincdb file error %s", err)
			return
		}

	}

	if data, err = ioutil.ReadFile(plugin.SincePath); err != nil {
		utils.Logger.Errorf("Read sincedb file error %s", err)
		return
	}

	if err = json.Unmarshal(data, &plugin.SinceDBInfos); err != nil {
		utils.Logger.Errorf("ReUnmarshal sincedb file error %s", err)
		return
	}

	return
}

// save since data info.
func (plugin *PluginConfig) saveSinceDB() (err error) {
	var (
		data []byte
	)
	plugin.SinceLastSaveTime = time.Now()

	if plugin.SincePath == "" || plugin.SincePath == "/dev/null" {
		utils.Logger.Warnf("Sincedb path miss config")
		return
	}

	if data, err = json.MarshalIndent(plugin.SinceDBInfos, "", "\t"); err != nil {
		utils.Logger.Errorf("Marshal sincedb failed: %s", err)
		return
	}
	plugin.sinceLastInfos = data

	if err = ioutil.WriteFile(plugin.SincePath, data, 0664); err != nil {
		utils.Logger.Errorf("Write sincedb failed: %s", err)
		return
	}

	return
}

// check since data info.
func (plugin *PluginConfig) checkAndSaveSinceDB() (err error) {
	var (
		data []byte
	)
	if time.Since(plugin.SinceLastSaveTime) > time.Duration(plugin.Intervals)*time.Second {
		if data, err = json.Marshal(plugin.SinceDBInfos); err != nil {
			utils.Logger.Errorf("Marshal sincedb failed: %s", err)
			return
		}
		if bytes.Compare(data, plugin.sinceLastInfos) != 0 {
			err = plugin.saveSinceDB()
		}
	}
	return
}

// watch log files and emit logevent.
func (plugin *PluginConfig) watch(inchan utils.InChan) (err error) {
	defer func() {
		if err != nil {
			utils.Logger.Errorf("File input plugin watch error %s", err)
		}
	}()

	var (
		allfiles = make([]string, 0)
		fi       os.FileInfo
	)

	if err = plugin.loadSinceDB(); err != nil {
		utils.Logger.Errorf("loadSinceDB return error %s", err)
		return
	}

	if len(plugin.DirsPath) < 1 {
		utils.Logger.Errorf("No director need to watch.")
		return
	}

	// find all log file path.
	for _, dir := range plugin.DirsPath {
		fl, err := utils.FileList(dir, plugin.FileType)
		if err != nil {
			utils.Logger.Errorln(err)
		}
		allfiles = append(allfiles, fl...)
	}

	// loop save sincdb
	go func() {
		plugin.wgExit.Add(1)
		defer plugin.wgExit.Done()

		for plugin.running {
			time.Sleep(time.Duration(plugin.Intervals) * time.Second)
			if err = plugin.checkAndSaveSinceDB(); err != nil {
				return
			}
		}
	}()

	for _, fp := range allfiles {
		// get all sysmlinks.
		if fp, err = filepath.EvalSymlinks(fp); err != nil {
			utils.Logger.Warnf("Get symlinks failed: %s error %s", fp, err)
			continue
		}
		// check file status.
		if fi, err = os.Stat(fp); err != nil {
			utils.Logger.Warnf("Get file  status %s error %s", fp, err)
			continue
		}
		// skip directory
		if fi.IsDir() {
			utils.Logger.Warnf("Skipping directory %s", fi.Name())
			continue
		}
		// monitor file.
		utils.Logger.Info("Watching ", fp)
		readEventChan := make(chan fsnotify.Event, 10)
		go plugin.loopRead(readEventChan, fp, inchan)
		go plugin.loopWatch(readEventChan, fp, fsnotify.Create|fsnotify.Write)
	}

	return
}

// loopRead
func (plugin *PluginConfig) loopRead(
	readEventChan chan fsnotify.Event,
	realPath string,
	inchan utils.InChan,
) (err error) {
	var (
		since     *SinceDBInfo
		fp        *os.File
		truncated bool
		ok        bool
		whence    int // see File.Seek
		reader    *bufio.Reader
		line      string
		size      int

		buffer = &bytes.Buffer{}
	)

	// for stopping
	plugin.wgExit.Add(1)
	defer plugin.wgExit.Done()

	// check and set the sincdb
	if since, ok = plugin.SinceDBInfos[realPath]; !ok {
		plugin.SinceDBInfos[realPath] = &SinceDBInfo{}
		since = plugin.SinceDBInfos[realPath]
	}
	// set or get offset index.
	if since.Offset == 0 {
		if plugin.Follow {
			whence = os.SEEK_END // seek relative to the end
		} else {
			whence = os.SEEK_SET // seek relative to the origin of the file
		}
	} else {
		whence = os.SEEK_SET // seek relative to the origin of the file
	}
	if fp, reader, err = openFileAt(realPath, since.Offset, whence); err != nil {
		return
	}
	defer fp.Close()
	// seek beginning.
	if truncated, err = isTruncated(fp, since); err != nil {
		return
	}
	if truncated {
		utils.Logger.Warnf("File truncated, seeking to beginning: %q", realPath)
		since.Offset = 0
		// change cursor
		if _, err = fp.Seek(0, os.SEEK_SET); err != nil {
			utils.Logger.Errorf("seek file failed: %q", realPath)
			return
		}
	}
	// looping read and check file change.
	for plugin.running {
		if line, size, err = readLine(reader, buffer); err != nil {
			if err == io.EOF {
				// wait incomming log message
				watchev := <-readEventChan
				if watchev.Name == "@@@exit" {
					return
				}

				if watchev.Op&fsnotify.Create == fsnotify.Create {
					// log file rollover.
					fp.Close()
					since.Offset = 0
					if fp, reader, err = openFileAt(realPath, 0, os.SEEK_SET); err != nil {
						return
					}
				}
				if truncated, err = isTruncated(fp, since); err != nil {
					return
				}
				if truncated {
					since.Offset = 0
					if _, err = fp.Seek(0, os.SEEK_SET); err != nil {
						return
					}
					continue
				}
				continue
			} else {
				return
			}
		}

		event := utils.LogEvent{
			Timestamp: time.Now(),
			Message:   line,
			Extra: map[string]interface{}{
				"host":   plugin.hostname,
				"path":   realPath,
				"offset": since.Offset,
				"size":   size,
			},
		}
		since.Offset += int64(size)
		// push log event to the pipeline.
		inchan <- event
		plugin.checkAndSaveSinceDB()
	}

	return
}

// loopWatch
func (plugin *PluginConfig) loopWatch(readEventChan chan fsnotify.Event, realPath string, op fsnotify.Op) (err error) {
	var (
		event fsnotify.Event
	)

	// for stopping
	plugin.wgExit.Add(1)
	defer plugin.wgExit.Done()

	for plugin.running {
		// wait event and notify channel.
		if event, err = waitWatchEvent(realPath, op, plugin.dbInfosLock); err != nil {
			readEventChan <- fsnotify.Event{
				Name: "@@@exit",
				Op:   0,
			}
			return
		}
		readEventChan <- event
	}
	return
}

// isTruncated check file is truncated or not
func isTruncated(fp *os.File, since *SinceDBInfo) (truncated bool, err error) {
	var (
		fi os.FileInfo
	)
	if fi, err = fp.Stat(); err != nil {
		return
	}
	// Old offset larger than file size.
	if fi.Size() < since.Offset {
		truncated = true
	} else {
		truncated = false
	}
	return
}

// openFile open a file move cursor to offset
func openFileAt(realPath string, offset int64, whence int) (fp *os.File, reader *bufio.Reader, err error) {
	if fp, err = os.Open(realPath); err != nil {
		return
	}

	if _, err = fp.Seek(offset, whence); err != nil {
		err = errors.New("seek file failed: " + realPath)
		return
	}

	reader = bufio.NewReaderSize(fp, 16*1024)
	return
}

// readLine using bufio.Reader.ReadLine, just handler the isPrefix value.
func readLine(reader *bufio.Reader, buffer *bytes.Buffer) (line string, size int, err error) {
	var (
		segment []byte
		next    bool
	)
	// if link file to stdin, can block forever here.
	for {
		if segment, next, err = reader.ReadLine(); err != nil {
			if err != io.EOF {
				err = errors.New("read line failed")
			}
			return
		}
		if _, err = buffer.Write(segment); err != nil {
			return
		}

		if next {
			continue
		} else {
			size = buffer.Len()
			line = buffer.String()
			//fix delim
			reader.UnreadByte()
			reader.UnreadByte()
			var b []byte
			b, err = reader.ReadBytes('\n')
			if err != nil {
				return
			}
			if b[0] == '\r' {
				size++
			}
			size++
			// clear buffer
			buffer.Reset()
			return
		}
	}
}

// waitWatchEvent using fsnotify directory watcher.
func waitWatchEvent(realPath string, op fsnotify.Op, lock *sync.RWMutex) (event fsnotify.Event, err error) {
	var (
		dir     string
		watcher *fsnotify.Watcher
		ok      bool
	)

	dir = filepath.Dir(realPath)
	// one dir on watcher, use the existied watcher.
	func() {
		lock.Lock()
		defer lock.Unlock()
		if watcher, ok = mapWatcher[dir]; !ok {
			if watcher, err = fsnotify.NewWatcher(); err != nil {
				err = errors.New("fsnotify create new watcher failed: " + dir)
				return
			}
			if err = watcher.Add(dir); err != nil {
				err = errors.New("add new watch path failed: " + dir)
				return
			}
			mapWatcher[dir] = watcher
		}
	}()

	for {
		select {
		case event = <-watcher.Events:
			if event.Name == realPath {
				if op > 0 {
					// if this is create or write event
					if event.Op&op > 0 {
						return
					}
				} else {
					// any type of event
					return
				}
			}
		case err = <-watcher.Errors:
			err = errors.New("watcher error " + realPath)
			return
		}
	}
}
