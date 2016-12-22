// +build integration
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
	"config2consul/docker_compose"
	consulapi "github.com/hashicorp/consul/api"
	. "github.com/smartystreets/goconvey/convey"
	"net"
	"net/http"
	"testing"
	"time"
	"errors"
	"path/filepath"
	"bytes"
	"io"
	"crypto/tls"
	"config2consul/log"
)

func checkIfListenningOnPort(address string) bool {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func getHttpResponse(url string) (resp *http.Response, dfrFunc func(), err error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err = client.Get(url)
	dfrFunc = func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}
	return resp, dfrFunc, err
}

func checkIfHttpResponceNotEqual(url string, unwanted string) bool {
	resp, dfrFunc, err := getHttpResponse(url)
	if dfrFunc != nil {
		defer dfrFunc()
	}
	if err != nil {
		return false
	}

	if resp.StatusCode != 200 {
		return false
	}
	var bodyBuf bytes.Buffer
	if _, err := io.Copy(&bodyBuf, resp.Body); err != nil {
		log.Errorf("ERROR: %v", err)
		return false
	}
	response := bodyBuf.String()
	log.Debugf("HTTP response: '%s'", response)
	if response == unwanted {
		log.Error("HTTP Response is empty")
		return false
	}
	return true
}

func TestInjestConsul(t *testing.T) {
	log.SetLevel(log.PanicLevel)

	consul, deferFn, err := createTestProject("../testing/integration/consul_base/docker-compose.yml", "ssl/ca.crt", "ssl/consul_client.crt", "ssl/consul_client.key")
	if err != nil {
		t.Fatal(err)
	}
	defer deferFn()

	Convey("Controlling KV entries", t, func() {
		Convey("A KV entry is created", func() {
			keyPath := "aa/blah"
			value := "boom"
			configData := consulConfig{
				KeyValue: map[string]string{
					keyPath: value,
				},
			}
			importConfig(consul, &configData)

			result := GetValue(t, consul, keyPath)

			So(string(result.Value), ShouldEqual, value)
		})

		Convey("A KV is overwritten if existed", func() {
			keyPath := "bb/blah"
			value := "foo"

			CreateKV(t, consul, keyPath, []byte("bar"))

			configData := consulConfig{
				KeyValue: map[string]string{
					keyPath: value,
				},
			}
			importConfig(consul, &configData)

			result := GetValue(t, consul, keyPath)

			So(string(result.Value), ShouldEqual, value)
		})

		Convey("A KV is deleted if not defined by a policy", func() {
			keyPath := "cc/blah"

			CreateKV(t, consul, keyPath, []byte("foo"))

			configData := consulConfig{
				KeyValue: map[string]string{
					"dd/blah": "test",
				},
			}

			importConfig(consul, &configData)

			result := GetValue(t, consul, keyPath)

			So(result, ShouldBeNil)
		})

		Convey("A KV tree is ignored if said so", func() {
			keyPath := "ee/blah"
			value := "foo"

			CreateKV(t, consul, keyPath, []byte(value))

			configData := consulConfig{
				KeyValue: map[string]string{
					"ee/": "${ignore}",
				},
			}
			importConfig(consul, &configData)

			result := GetValue(t, consul, keyPath)

			So(string(result.Value), ShouldEqual, value)
		})
	})

	Convey("Controlling ACLs", t, func() {
		Convey("ACL is created", func() {
			aclName := "test"
			aclType := "management"
			aclRules := `# Default all keys to read-only
key "network/" {
	policy = "write"
}
key "" {
	policy = "deny"
}`

			config.Conf.PreserveBuiltInTokens = true
			configData := consulConfig{
				Policies: acls{
					acl{
						Name:  aclName,
						Type:  aclType,
						Rules: aclRules,
					},
				},
			}
			err := importConfig(consul, &configData)
			So(err, ShouldBeNil)

			aclz := GetAclByName(t, consul, aclName)

			So(aclz, ShouldNotBeNil)
			So(string(aclz.Type), ShouldEqual, aclType)
			So(string(aclz.Rules), ShouldEqual, aclRules)
		})

		Convey("ACL duplicate is found and rejected", func() {
			aclName := "test bbb"
			aclType := "client"
			aclRules := `# test`

			config.Conf.PreserveBuiltInTokens = true
			configData := consulConfig{
				Policies: acls{
					acl{
						Name:  aclName,
						Type:  aclType,
						Rules: aclRules,
					},
					acl{
						Name:  aclName,
						Type:  aclType,
						Rules: aclRules,
					},
				},
			}
			err := importConfig(consul, &configData)

			So(err, ShouldNotBeNil)
		})

		Convey("ACL updated", func() {
			aclName := "test bbb"
			aclType := "client"
			aclRules := `# test`

			CreateACL(t, consul, aclName, aclType, "# empty")

			config.Conf.PreserveBuiltInTokens = true
			configData := consulConfig{
				Policies: acls{
					acl{
						Name:  aclName,
						Type:  aclType,
						Rules: aclRules,
					},
				},
			}
			err := importConfig(consul, &configData)
			So(err, ShouldBeNil)

			aclz := GetAclByName(t, consul, aclName)
			So(aclz, ShouldNotBeNil)
			So(string(aclz.Type), ShouldEqual, aclType)
			So(string(aclz.Rules), ShouldEqual, aclRules)
		})

		Convey("Multiple ACLs with the same name exist", func() {
			aclName := "test ccc"
			aclType := "client"
			aclRules := `# test`

			CreateACL(t, consul, aclName, aclType, "# rules 1")
			CreateACL(t, consul, aclName, aclType, "# rules 2")

			config.Conf.PreserveBuiltInTokens = true
			configData := consulConfig{
				Policies: acls{
					acl{
						Name:  aclName,
						Type:  aclType,
						Rules: aclRules,
					},
				},
			}
			err := importConfig(consul, &configData)
			So(err, ShouldNotBeNil)
		})

		Convey("ACL is ignored if marked as such", func() {
			aclName := "test ddd"
			aclType := "client"
			aclRules := `# test`

			CreateACL(t, consul, aclName, aclType, aclRules)
			configData := consulConfig{
				Policies: acls{
					acl{
						Name:  "Master Token",
						Rules: "${ignore}",
					},
					acl{
						Name:  "Anonymous Token",
						Rules: "${ignore}",
					},
					acl{
						Name:  aclName,
						Type:  aclType,
						Rules: "${ignore}",
					},
				},
			}
			err := importConfig(consul, &configData)
			So(err, ShouldNotBeNil)

			aclz := GetAclByName(t, consul, aclName)
			So(aclz, ShouldNotBeNil)
			So(string(aclz.Type), ShouldEqual, aclType)
			So(string(aclz.Rules), ShouldEqual, aclRules)
		})

	})
}

type consulTestClient struct {
	address string
	client  *consulapi.Client
}

func createTestProject(projectPath string, CaFile string, CertFile string, KeyFile string) (*consulClient, func(), error) {
	projectName := "testproject"

	project, err := docker_compose.NewDockerComposeProjectFromFile(projectName, projectPath)
	if err != nil {
		return nil, nil, err
	}
	connection, deferFn, err := project.Up()
	if err != nil {
		log.Fatalf("Failed to start docker project: %s", err)
		return nil, deferFn, err
	}
	log.Debugf("Connection: %s", connection)

	// check if Consul container up
	if running, _ := docker_compose.IsRunning(projectName, "consul"); !running {
		log.Fatalf("Consul Container is not running. Aborting ...")
		return nil, deferFn, errors.New("Container is not running. Aborting ...")
	}

	// TODO: define an exit timeout
	// TODO: externalize ports and scheme
	for ok := false; !ok; ok = checkIfHttpResponceNotEqual("https://" + connection + ":8501/v1/status/leader", "\"\"") {
		log.Debug("Waiting on HTTP connection https://" + connection + ":8501/v1/status/leader")
		time.Sleep(500 * time.Millisecond)
	}

	consul := consulClient{}
	dir := filepath.Dir(projectPath)

	consul.Client = createClient(connection+":8501", "https", "a49e7360-f150-463a-9a29-3eb186ffae1a", filepath.Join(dir, CaFile), filepath.Join(dir, CertFile), filepath.Join(dir, KeyFile))

	return &consul, deferFn, nil
}

func CreateKV(t *testing.T, consul *consulClient, keyPath string, value []byte) {
	w := consulapi.WriteOptions{}
	kv := consulapi.KVPair{
		Key:   keyPath,
		Value: value,
	}
	_, err := consul.Client.KV().Put(&kv, &w)
	if err != nil {
		t.Fatal(err)
	}
}

func GetValue(t *testing.T, consul *consulClient, keyPath string) *consulapi.KVPair {
	q := consulapi.QueryOptions{}
	result, _, err := consul.Client.KV().Get(keyPath, &q)
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func GetAclByName(t *testing.T, consul *consulClient, name string) *consulapi.ACLEntry {
	q := consulapi.QueryOptions{}
	result, _, err := consul.Client.ACL().List(&q)
	if err != nil {
		t.Fatal(err)
	}

	for _, acl := range result {
		if acl.Name == name {
			return acl
		}
	}

	return nil
}

func GetAclById(t *testing.T, consul *consulClient, id string) *consulapi.ACLEntry {
	q := consulapi.QueryOptions{}

	result, _, err := consul.Client.ACL().Info(id, &q)
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func DumpACLs(consul *consulClient) {
	log.Debug("Dumping all ACLs")
	q := consulapi.QueryOptions{}
	acls, _, _ := consul.Client.ACL().List(&q)
	for _, acl := range acls {
		log.Debugf("Found ACL %s:%s", acl.ID, acl.Name)
	}
}

func CreateACL(t *testing.T, consul *consulClient, aclName string, aclType string, value string) {
	w := consulapi.WriteOptions{}
	newAcl := consulapi.ACLEntry{
		Name: aclName,
		// Type is either client or management
		Type:  aclType,
		Rules: value,
	}
	id, _, err := consul.Client.ACL().Create(&newAcl, &w)
	if err != nil {
		t.Fatal(err)
	}
	log.Debug("Created ACL with ID: " + id)
}
