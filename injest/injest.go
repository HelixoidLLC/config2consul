/*
 * Copyright 2016 Igor Moochnick
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package injest

import (
	"config2consul/config"
	log "github.com/Sirupsen/logrus"
	"github.com/golang/glog"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-cleanhttp"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
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
		Policies: acls{},
		KeyValue: make(map[string]string),
	}

	filename, _ := filepath.Abs(path)
	fileInfo, _ := os.Stat(filename)
	if fileInfo.IsDir() {
		// TODO: read only files with *.yml extension
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
	log.Info("Loading file: " + filename)
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

func ImportConfig(consConf *consulConfig) {
	consul := create(&config.Conf)
	importConfig(consul, consConf)
	consul.Client = nil
}

func importConfig(consul *consulClient, config *consulConfig) error {
	if len(config.Policies) > 0 {
		err := consul.importPolicies(&config.Policies)
		if err != nil {
			return err
		}
	} else {
		glog.Info("No ACLs to import.")
	}
	if len(config.KeyValue) > 0 {
		err := consul.importKeyValue(&config.KeyValue)
		if err != nil {
			return err
		}
	} else {
		glog.Info("No KVs to import.")
	}
	return nil
}

func create(config *config.Config) *consulClient {
	consul := consulClient{}

	consul.Client = createClient(config.Address, config.Scheme, config.Token, config.CaFile, config.CertFile, config.KeyFile)

	return &consul
}

func createClient(address string, scheme string, token string, CaFile string, CertFile string, KeyFile string) *consulapi.Client {
	//consul := consulClient{}

	config := consulapi.DefaultConfig()
	config.Address = address
	config.Token = token

	if scheme == "https" {
		config.Scheme = "https"
		config.HttpClient.Transport = createTlsTransport(CaFile, CertFile, KeyFile)
	}

	// Get a new client
	client, err := consulapi.NewClient(config)
	if err != nil {
		glog.Fatal("Can't connect to consul")
	}

	return client
}

func createTlsTransport(CAFile string, CertFile string, KeyFile string) http.RoundTripper {

	tlsClientConfig, err := consulapi.SetupTLSConfig(&consulapi.TLSConfig{
		InsecureSkipVerify: true,
		CAFile:             CAFile,
		CertFile:           CertFile,
		KeyFile:            KeyFile,
	})

	if err != nil {
		panic(err)
	}

	transport := cleanhttp.DefaultPooledTransport()
	transport.TLSClientConfig = tlsClientConfig
	return transport
}
