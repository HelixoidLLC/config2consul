package injest

import (
	"config2consul/log"
	consulapi "github.com/hashicorp/consul/api"
	"testing"
)

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
