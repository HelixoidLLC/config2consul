package injest

import (
	"config2consul/config"
	"errors"
	consulapi "github.com/hashicorp/consul/api"
	"strings"
	log "github.com/Sirupsen/logrus"
	"fmt"
)

func (consul *consulClient) getCurrentAcls1() (*map[string]string, error) {
	currentAcls := make(map[string]string)

	q := consulapi.QueryOptions{}
	aclEntry, _, err := consul.Client.ACL().List(&q)
	if err != nil {
		return &currentAcls, err
	}

	for _, acl := range aclEntry {
		//if _, ok := currentAcls[acl.Name]; ok {
		//	glog.Warning("Found duplicate ACL name: " + acl.Name)
		//}
		currentAcls[acl.ID] = acl.Name
	}

	return &currentAcls, nil
}

func (consul *consulClient) importPolicies(newACLs *acls) error {

	currentAcls1, _ := consul.getCurrentAcls1()

	// Do nothing if no ACLs were found
	if len(*newACLs) == 0 {
		return nil
	}

	// Validate there are no duplicates in existing values
	uniqueValues := make(map[string]string)
	for id, name := range *currentAcls1 {
		if id2, ok := uniqueValues[name]; ok {
			err := fmt.Sprintf("Found existing Policies with name '%s'. ids: (%s, %s)", name, id, id2)
			log.Errorf(err)
			return errors.New(err)
		}
		uniqueValues[name] = id
		log.Debugf("Found ACL %s:%s", id, name)
	}

	// Validate there are no duplicates in porovided values
	newACLmap := make(map[string]acl)
	for _, acl := range *newACLs {
		if _, seen := newACLmap[acl.Name]; seen {
			log.Errorf("Found duplicate ACL ID '%s' in the injest. Aborting ...", acl.Name)
			return errors.New("Found duplicate ACL ID '" + acl.Name + "' in the injest.")
		}
		newACLmap[acl.Name] = acl
	}

	// Injest ACLs
	for _, acl := range *newACLs {
		done, _ := consul.applyAcl(&acl, &uniqueValues)
		if done {
			delete(uniqueValues, acl.Name)
		}
	}

	if config.Conf.PreserveBuiltInTokens {
		log.Info("Preserving Master and Anonymous Token")
		delete(uniqueValues, "Master Token")
		delete(uniqueValues, "Anonymous Token")
	}

	// TODO: add ${ignore} rules for ACLs prefixes
	if config.Conf.PreserveVaultACLs {
		keys_to_save := []string{}
		for key := range uniqueValues {
			if strings.HasPrefix(key, "Vault ") {
				keys_to_save = append(keys_to_save, key)
				log.Info("Preserving Vault ACL: " + key)
				delete(uniqueValues, key)
			}
		}
		keys_to_save = nil
	}

	// Purging the rest of the values
	for name, id := range uniqueValues {
		log.Warningf("Deleting unexpected ACL '%s' with ID: %s", name, id)
		consul.deleteAcl(id)
	}

	return nil
}

func (consul *consulClient) applyAcl(acl *acl, currentAcls *map[string]string) (bool, error) {
	w := consulapi.WriteOptions{}

	if acl.Rules == "${ignore}" {
		log.Infof("Ignoring %s ACL", acl.Name)
		return true, nil
	}

	if id, ok := (*currentAcls)[acl.Name]; ok {
		q := consulapi.QueryOptions{}
		// TODO: is it by the name or ID, or both?
		existingAcl, _, err := consul.Client.ACL().Info(id, &q)
		if err != nil {
			log.Errorf("Failed to get info for ACL w/ID: %s. %v", id, err)
			return false, err
		}
		if acl.Type == "" {
			acl.Type = "client"
		}
		if existingAcl.Type == acl.Type && existingAcl.Rules == acl.Rules {
			log.Infof("Skipping ACL '%s' with ID: %s. Nothing to update.", acl.Name, existingAcl.ID)
			return true, nil
		}
		existingAcl.Rules = acl.Rules
		existingAcl.Type = acl.Type
		log.Infof("Updating ACL '%s' with ID: %s", acl.Name, existingAcl.ID)
		_, err = consul.Client.ACL().Update(existingAcl, &w)
		if err != nil {
			log.Errorf("Failed to update ACL. %v", err)
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
		log.Errorf("Failed to create ACL w/Name: %s. %v", acl.Name, err)
		return false, err
	}
	log.Infof("A new ACL '%s' has been created with ID: %s", acl.Name, id)
	return true, nil
}

func (consul *consulClient) deleteAcl(id string) (bool, error) {
	w := consulapi.WriteOptions{}
	_, err := consul.Client.ACL().Destroy(id, &w)
	if err != nil {
		log.Errorf("Failed to delete ACL w/ID: %s. %v", id, err)
		return false, err
	}
	return true, nil
}
