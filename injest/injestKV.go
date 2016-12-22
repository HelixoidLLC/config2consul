package injest

import (
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/golang/glog"
	consulapi "github.com/hashicorp/consul/api"
	"strings"
)

func (consul *consulClient) importKeyValue(keyValue *map[string]string) error {
	q := consulapi.QueryOptions{}
	// TODO: preserve more information, like "Index"
	currentKvPairsOrig, _, _ := consul.Client.KV().List("", &q)
	currentKvPairs := map[string]string{}
	for _, kv := range currentKvPairsOrig {
		glog.Infof("%s: %d", kv.Key, kv.CreateIndex)
		// TODO: Value is byte[] - deal with it
		currentKvPairs[kv.Key] = string(kv.Value)
	}

	for key, value := range *keyValue {
		if key[len(key)-1] == '/' {
			if value == "${ignore}" {
				glog.Info("Ignoring tree: " + key)
				for k := range currentKvPairs {
					if strings.HasPrefix(k, key) {
						delete(currentKvPairs, k)
					}
				}
			} else {
				glog.Error("Unexpected value for the key tree: " + key)
				return errors.New("Unexpected value the key tree: " + key)
			}
		} else {
			var done bool
			if value == "${ignore}" {
				done = true
			} else {
				done, _ = consul.applyKV(key, value, &currentKvPairs)
			}
			if done {
				delete(currentKvPairs, key)
			}
		}
	}

	logrus.Info("Need to delete what wasn't defined")
	for key := range currentKvPairs {
		logrus.Warningf("Deleting unexpected Key '%s'", key)
		consul.deleteKV(key)
	}

	return nil
}

func (consul *consulClient) applyKV(key string, value string, currentKVList *map[string]string) (bool, error) {
	w := consulapi.WriteOptions{}

	if currentValue, ok := (*currentKVList)[key]; ok {
		if value == "${ignore}" || value == currentValue {
			return true, nil
		}

		glog.Warningf("Found unexpected value of key %s. Overwriting ...", key)

		kv := consulapi.KVPair{
			Key:   key,
			Value: []byte(value),
		}
		_, err := consul.Client.KV().Put(&kv, &w)
		if err != nil {
			glog.Error("Failed to update key: " + key)
			return false, errors.New("Failed to update key: " + key)
		}
		return true, nil
	}

	// TODO: DRI
	kv := consulapi.KVPair{
		Key:   key,
		Value: []byte(value),
	}
	_, err := consul.Client.KV().Put(&kv, &w)
	if err != nil {
		glog.Error("Failed to update key: " + key)
		return false, errors.New("Failed to update key: " + key)
	}

	return true, nil
}

func (consul *consulClient) deleteKV(key string) (bool, error) {
	w := consulapi.WriteOptions{}
	logrus.Info("Deleting key: " + key)
	_, err := consul.Client.KV().Delete(key, &w)
	if err != nil {
		logrus.Errorf("Failed to delete key: %s. %v", key, err)
		return false, err
	}
	return true, nil
}
