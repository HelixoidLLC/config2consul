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
	"config2consul/log"
	"errors"
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"strings"
)

func (consul *consulClient) importKeyValue(keyValue *map[string]interface{}) error {
	q := consulapi.QueryOptions{}
	// TODO: preserve more information, like "Index"
	currentKvPairsOrig, _, _ := consul.Client.KV().List("", &q)
	currentKvPairs := make(map[string]string)
	for _, kv := range currentKvPairsOrig {
		log.Infof("Found %s: %d", kv.Key, kv.CreateIndex)
		currentKvPairs[kv.Key] = string(kv.Value)
	}

	err := consul.importTree(keyValue, currentKvPairs)
	if err != nil {
		return err
	}

	if len(currentKvPairs) > 0 {
		log.Infof("Deleting %d runaway key pairs", len(currentKvPairs))
		for key := range currentKvPairs {
			log.Warningf("Deleting runaway Key '%s'", key)
			consul.deleteKV(key)
		}
	}

	return nil
}

func (consul *consulClient) importTree(keyValue *map[string]interface{}, currentKvPairs map[string]string) error {

	for key, i_value := range *keyValue {
		if len(key) == 0 {
			log.Error("Got empty key in the K/V collection.")
			continue
		}
		if key[len(key)-1] == '/' {
			switch value := i_value.(type) {
			case string:
				if value == "${ignore}" {
					log.Info("Ignoring tree: " + key)
					for k := range currentKvPairs {
						if strings.HasPrefix(k, key) {
							delete(currentKvPairs, k)
						}
					}
				} else {
					err_text := fmt.Sprintf("Unexpected string value for the key tree '%s' of type: %s", key, value)
					log.Error(err_text)
					return errors.New(err_text)
				}
			case map[interface{}]interface{}:

				map_value := convert_map(&value, key)

				log.Debugf("Importing tree %s", key)
				consul.importTree(map_value, currentKvPairs)
			default:
				err_text := fmt.Sprintf("Unexpected value for the key tree '%s' of type: %T", key, i_value)
				log.Error(err_text)
				return errors.New(err_text)
			}
		} else {
			str_value, ok := get_string_value(i_value)
			if !ok {
				err_text := fmt.Sprintf("Unexpected value for the key '%s': %T", key, i_value)
				log.Error(err_text)
				return errors.New(err_text)
			}

			var done bool
			if str_value == "${ignore}" {
				done = true
			} else {
				done, _ = consul.applyKV(key, str_value, &currentKvPairs)
			}
			if done {
				delete(currentKvPairs, key)
			}
		}
	}

	return nil
}

func get_string_value(value interface{}) (string, bool) {
	str_value := ""
	switch value := value.(type) {
	case string:
		str_value = value
	case *string:
		str_value = *value
	default:
		return "", false
	}

	return str_value, true
}

func convert_map(input *map[interface{}]interface{}, path_prefix string) *map[string]interface{} {
	output := make(map[string]interface{})

	for key, value := range *input {
		switch key := key.(type) {
		case string:
			key_path := path_prefix + key
			switch value := value.(type) {
			case string:
				output[key_path] = value
			default:
				output[key_path+"/"] = value
			}
		}
	}

	return &output
}

func (consul *consulClient) applyKV(key string, value string, currentKVList *map[string]string) (bool, error) {
	w := consulapi.WriteOptions{}

	if currentValue, ok := (*currentKVList)[key]; ok {
		if value == "${ignore}" || value == currentValue {
			return true, nil
		}

		log.Warningf("Value of key %s has been changed. Overwriting ...", key)
	}

	kv := consulapi.KVPair{
		Key:   key,
		Value: []byte(value),
	}
	_, err := consul.Client.KV().Put(&kv, &w)
	if err != nil {
		err_text := fmt.Sprintf("#2 Failed to update key '%s' with value '%s'. %#v", key, value, err)
		log.Error(err_text)
		return false, errors.New(err_text)
	}

	return true, nil
}

func (consul *consulClient) deleteKV(key string) (bool, error) {
	w := consulapi.WriteOptions{}
	log.Info("Deleting key: " + key)
	_, err := consul.Client.KV().Delete(key, &w)
	if err != nil {
		log.Errorf("Failed to delete key: %s. %v", key, err)
		return false, err
	}
	return true, nil
}
