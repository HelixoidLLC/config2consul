package config

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/golang/glog"
	"io/ioutil"
)

// Config represents the configuration information.
type Config struct {
	//Debug bool   `json:"debug"`
	//Token string `json:"token"`
	Path    string `json:"path,omitempty"`
	Address string `json:"address,omitempty"`
	Scheme  string `json:"scheme,omitempty"`
	Token   string `json:"token,omitempty"`

	CaFile   string `json:"ca_file,omitempty"`
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`

	PreserveMasterToken bool `json:"preserve_master_token,omitempty"`
	PreserveVaultACLs   bool `json:"preserve_vault_acls,omitempty"`

	PreserveExistingKV bool `json:"preserve_vault_acls,omitempty"`
}

// Conf contains the initialized configuration struct
var Conf Config

//func init() {
//	flag.BoolVar(&logging.toStderr, "logtostderr", false, "log to standard error instead of files")
//	flag.BoolVar(&logging.alsoToStderr, "alsologtostderr", false, "log to standard error as well as files")
//	flag.Var(&logging.verbosity, "v", "log level for V logs")
//	flag.Var(&logging.stderrThreshold, "stderrthreshold", "logs at or above this threshold go to stderr")
//	flag.Var(&logging.vmodule, "vmodule", "comma-separated list of pattern=N settings for file-filtered logging")
//	flag.Var(&logging.traceLocation, "log_backtrace_at", "when logging hits line file:N, emit a stack trace")
//}

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "./config.json", "path to the config file")
}

func ReadConfig() error {
	// Get the config file
	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return errors.New("Cant load config file at path: " + configPath)
	}
	err = json.Unmarshal(configFile, &Conf)
	if err != nil {
		glog.Errorf("Failed to load config file: %v", err)
	}

	return nil
}

/*
func GetValue(path string, key string) (string, error) {
	parts := strings.Split(path, "/")
	location := Conf.Raw
	for _, part := range parts {
		if _, ok := location[part]; !ok {
			return "", errors.New("Path wasn't found")
		}
		location = location[part].(map[string]interface{})
	}

	if value, ok := location[key]; ok {
		return value.(string), nil
	}

	return "", errors.New("Key wasn't found")
}

func GetValueWithDefault(path string, key string, defaultValue string) string {
	value, err := GetValue(path, key)
	if err != nil {
		return defaultValue
	}

	return value
}
*/