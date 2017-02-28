package utils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var (
	agentName = ""
)

// CmdAgentName set the agent name.
func CmdAgentName(name string) error {
	if agentName != "" || name == "" {
		return errors.New("")
	}
	agentName = name
	return nil
}

// CmdEtcdHost load config from etcd.
func CmdEtcdHost(endpoints string) (confs []Config, err error) {
	eps, err := cmdEndpointList(endpoints, ";")
	if err != nil {
		return
	}
	fmt.Println(eps)
	return
}

// CmdLocalPath load single file or *.json
func CmdLocalPath(pathDir string) (confs []Config, err error) {
	fileInfo, err := os.Stat(pathDir)
	if err != nil {
		return
	}
	if fileInfo.IsDir() {
		files, _ := cmdFileList(pathDir, "json")
		for _, f := range files {
			conf, err := LoadFromFile(f)
			if err != nil {
				Logger.Errorf("Can't load config file %s", f)
				continue
			} else {
				confs = append(confs, conf)
			}
		}
	} else {
		conf, err := LoadFromFile(pathDir)
		if err == nil {
			confs = append(confs, conf)
		}
	}
	return
}

// CmdRun start process.
func CmdRun(confs []Config) (err error) {
	for _, conf := range confs {
		if err = conf.RunInputs(); err != nil {
			return
		}

		if err = conf.RunFilters(); err != nil {
			return
		}

		if err = conf.RunOutputs(); err != nil {
			return
		}
	}
	return
}

// FileList export cmdFileList
func FileList(dirPath string, suffix string) ([]string, error) {
	return cmdFileList(dirPath, suffix)
}

// list all templates file in dirPath with suffix(json).
func cmdFileList(dirPath string, suffix string) (files []string, err error) {
	files = make([]string, 0, 10)

	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	PthSep := string(os.PathSeparator)
	suffix = strings.ToUpper(suffix)

	for _, fi := range dir {
		if fi.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			files = append(files, dirPath+PthSep+fi.Name())
		}
	}

	return files, nil
}

// Separate endpoint list.
func cmdEndpointList(endpoints string, sep string) (eps []string, err error) {
	eps = strings.Split(endpoints, sep)
	for i, v := range eps {
		eps[i] = strings.TrimSpace(v)
	}
	if len(eps) == 0 {
		err = errors.New("illegal etcd endpoints")
	}
	return
}
