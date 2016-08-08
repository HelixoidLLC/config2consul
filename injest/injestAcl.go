package injest

import (
	"config2consul/config"
	"errors"
	"github.com/golang/glog"
	consulapi "github.com/hashicorp/consul/api"
	"strings"
)

func (consul *consulClient) getCurrentAcls() (*map[string]string, error) {
	currentAcls := make(map[string]string)

	q := consulapi.QueryOptions{}
	aclEntry, _, err := consul.Client.ACL().List(&q)
	if err != nil {
		return &currentAcls, err
	}

	for _, acl := range aclEntry {
		// TODO: validate that Name is unique
		if _, ok := currentAcls[acl.Name]; ok {
			glog.Warning("Found duplicate ACL name: " + acl.Name)
		}
		currentAcls[acl.Name] = acl.ID
	}

	return &currentAcls, nil
}

func (consul *consulClient) importPolicies(newACLs *acls) {

	currentAcls, _ := consul.getCurrentAcls()

	// Do nothing if no ACLs were found
	if len(*newACLs) == 0 {
		return
	}

	newACLmap := make(map[string]acl)
	for _, acl := range *newACLs {
		if _, seen := newACLmap[acl.Name]; seen {
			glog.Errorf("Found duplicate ACL ID '%s' in the injest. Aborting ...", acl.Name)
			return
		}
		newACLmap[acl.Name] = acl
	}

	// TODO: validate there are no duplicates
	for _, acl := range *newACLs {
		done, _ := consul.applyAcl(&acl, currentAcls)
		if done {
			delete(*currentAcls, acl.Name)
		}
	}

	if config.Conf.PreserveMasterToken {
		glog.Info("Preserving Master Token")
		delete(*currentAcls, "Master Token")
	}

	// TODO: add ${ignore} rules for ACLs prefixes
	if config.Conf.PreserveVaultACLs {
		keys_to_save := []string{}
		for key := range *currentAcls {
			if strings.HasPrefix(key, "Vault ") {
				keys_to_save = append(keys_to_save, key)
				glog.Info("Preserving Vault ACL: " + key)
				delete(*currentAcls, key)
			}
		}
		keys_to_save = nil
	}

	for name, id := range *currentAcls {
		glog.Warningf("Deleting unexpected ACL '%s' with ID: %s", name, id)
		consul.deleteAcl(id)
	}
}

func (consul *consulClient) applyAcl(acl *acl, currentAcls *map[string]string) (bool, error) {
	w := consulapi.WriteOptions{}

	if name, ok := (*currentAcls)[acl.Name]; ok {
		q := consulapi.QueryOptions{}
		existingAcl, _, err := consul.Client.ACL().Info(name, &q)
		if err != nil {
			glog.Errorf("Failed to get info for ACL w/Name: %s. %v", name, err)
			return false, err
		}
		if acl.Type == "" {
			acl.Type = "client"
		}
		if existingAcl.Type == acl.Type && existingAcl.Rules == acl.Rules {
			glog.Infof("Skipping ACL '%s' with ID: %s. Nothing to update.", acl.Name, existingAcl.ID)
			return true, nil
		}
		existingAcl.Rules = acl.Rules
		existingAcl.Type = acl.Type
		glog.Infof("Updating ACL '%s' with ID: %s", acl.Name, existingAcl.ID)
		_, err = consul.Client.ACL().Update(existingAcl, &w)
		if err != nil {
			glog.Errorf("Failed to update ACL. %v", err)
			return false, errors.New("Failed to update ACL with Name: " + acl.Name)
		}

		return true, nil
	}

	newAcl := consulapi.ACLEntry{
		Name: acl.Name,
		// Type is either client or management
		Type:  "client",
		Rules: acl.Rules,
	}
	if acl.Type != "" {
		newAcl.Type = acl.Type
	}
	id, _, err := consul.Client.ACL().Create(&newAcl, &w)
	if err != nil {
		glog.Errorf("Failed to create ACL w/Name: %s. %v", acl.Name, err)
		return false, err
	}
	glog.Infof("A new ACL '%s' has been created with ID: %s", acl.Name, id)
	return true, nil
}

func (consul *consulClient) deleteAcl(id string) (bool, error) {
	w := consulapi.WriteOptions{}
	_, err := consul.Client.ACL().Destroy(id, &w)
	if err != nil {
		glog.Errorf("Failed to delete ACL w/ID: %s. %v", id, err)
		return false, err
	}
	return true, nil
}
