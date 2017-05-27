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
	"config2consul/log"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestInjestACL(t *testing.T) {
	log.SetLevel(log.PanicLevel)

	consul, deferFn, err := createTestProject("../testing/integration/consul_base/docker-compose.yml", "ssl/ca.crt", "ssl/consul_client.crt", "ssl/consul_client.key")
	if err != nil {
		t.Fatal(err)
	}
	defer deferFn()

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
