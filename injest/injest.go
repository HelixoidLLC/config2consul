package injest

import (
	"config2consul/config"
	"github.com/golang/glog"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-cleanhttp"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

type consulClient struct {
	Config *consulapi.Config
	Client *consulapi.Client
}

type acl struct {
	Name  string `json:"name"`
	Type  string `json:"type,omitempty"`
	Rules string `json:"rules,omitempty"`
}

type acls []acl

type consulConfig struct {
	Policies acls              `yaml:"policies,omitempty"`
	KeyValue map[string]string `yaml:"kv,omitempty"`
}

func ImportPath(path string) *consulConfig {

	masterConfig := consulConfig{
		Policies: []acl{},
		KeyValue: make(map[string]string),
	}

	filename, _ := filepath.Abs(path)
	fileInfo, _ := os.Stat(filename)
	if fileInfo.IsDir() {
		files, _ := ioutil.ReadDir(filename)
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			ImportFile(filepath.Join(filename, file.Name()), &masterConfig)
		}
	} else {
		ImportFile(path, &masterConfig)
	}

	return &masterConfig
}

func ImportFile(filename string, masterConfig *consulConfig) {
	glog.Info("Loading file: " + filename)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		glog.Fatal(err)
	}

	var config consulConfig

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		glog.Fatal(err)
	}

	masterConfig.mergeConfig(&config)
}

func (masterConfig *consulConfig) mergeConfig(newConfig *consulConfig) {
	(*masterConfig).Policies = append(masterConfig.Policies, newConfig.Policies...)
	for k, v := range newConfig.KeyValue {
		(*masterConfig).KeyValue[k] = v
	}
}

func ImportConfig(config *consulConfig) {
	consul := create()

	if len(config.Policies) > 0 {
		consul.importPolicies(&config.Policies)
	} else {
		glog.Info("No ACLs to import.")
	}
	if len(config.KeyValue) > 0 {
		consul.importKeyValue(&config.KeyValue)
	} else {
		glog.Info("No KVs to import.")
	}

	consul.Client = nil
}

func create() *consulClient {
	consul := consulClient{}

	consul.Config = consulapi.DefaultConfig()
	consul.Config.Address = config.Conf.Address
	consul.Config.Token = config.Conf.Token

	if config.Conf.Scheme == "https" {
		consul.Config.Scheme = "https"
		consul.configureTls()
	}

	// Get a new client
	client, err := consulapi.NewClient(consul.Config)
	if err != nil {
		glog.Fatal("Can't connect to consul")
	}

	consul.Client = client

	return &consul
}

func (c *consulClient) configureTls() {

	tlsClientConfig, err := consulapi.SetupTLSConfig(&consulapi.TLSConfig{
		InsecureSkipVerify: true,
		CAFile:             config.Conf.CaFile,
		CertFile:           config.Conf.CertFile,
		KeyFile:            config.Conf.KeyFile,
	})

	if err != nil {
		panic(err)
	}

	transport := cleanhttp.DefaultPooledTransport()
	transport.TLSClientConfig = tlsClientConfig
	c.Config.HttpClient.Transport = transport
}
