package config

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"path"
	"strings"
)

// Config implementation
// Implements the Config interface
type JsonConfig struct {
	obj map[string]interface{}
}

// FromJson reads the contents from the supplied reader.
// The content is parsed as json into a map[string]interface{}.
// It returns a JsonConfig struct pointer and any error encountered
func FromJson(reader io.Reader) (Config, error) {
	jsonBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &obj); err != nil {
		return nil, err
	}
	return &JsonConfig{obj}, nil
}

func FromJsonText(text string) (Config, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return nil, err
	}

	return &JsonConfig{obj}, nil
}

// GetString uses Get to fetch the value behind the supplied key.
// It returns a string with either the retreived value or the default value and any error encountered.
// If value is not a string it returns a UnexpectedValueTypeError
func (jconfig *JsonConfig) GetString(key string, defaultValue interface{}) (string, error) {
	configValue, err := jconfig.Get(key, defaultValue)
	if err != nil {
		return "", err
	}
	if stringValue, ok := configValue.(string); ok {
		return stringValue, nil
	} else {
		return "", &UnexpectedValueTypeError{key: key, value: configValue, message: "value is not a string"}
	}
}

// GetInt uses Get to fetch the value behind the supplied key.
// It returns a int with either the retreived value or the default value and any error encountered.
// If value is not a int it returns a UnexpectedValueTypeError
func (jconfig *JsonConfig) GetInt(key string, defaultValue interface{}) (int, error) {
	value, err := jconfig.GetFloat(key, defaultValue)
	if err != nil {
		return -1, err
	}
	return int(value), nil
}

// GetFloat uses Get to fetch the value behind the supplied key.
// It returns a float with either the retreived value or the default value and any error encountered.
// It returns a bool with either the retreived value or the default value and any error encountered.
// If value is not a float it returns a UnexpectedValueTypeError
func (jconfig *JsonConfig) GetFloat(key string, defaultValue interface{}) (float64, error) {
	configValue, err := jconfig.Get(key, defaultValue)
	if err != nil {
		return -1.0, err
	}
	if floatValue, ok := configValue.(float64); ok {
		return floatValue, nil
	} else if intValue, ok := configValue.(int); ok {
		return float64(intValue), nil
	} else {
		return -1.0, &UnexpectedValueTypeError{key: key, value: configValue, message: "value is not a float"}
	}
}

// GetBool uses Get to fetch the value behind the supplied key.
// It returns a bool with either the retreived value or the default value and any error encountered.
// If value is not a bool it returns a UnexpectedValueTypeError
func (jconfig *JsonConfig) GetBool(key string, defaultValue interface{}) (bool, error) {
	configValue, err := jconfig.Get(key, defaultValue)
	if err != nil {
		return false, err
	}
	if boolValue, ok := configValue.(bool); ok {
		return boolValue, nil
	} else {
		return false, &UnexpectedValueTypeError{key: key, value: configValue, message: "value is not a bool"}
	}
}

// GetAs uses Get to fetch the value behind the supplied key.
// The value is serialized into json and deserialized into the supplied target interface.
// It returns any error encountered.
func (jconfig *JsonConfig) GetAs(key string, target interface{}) error {
	configValue, err := jconfig.Get(key, nil)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(configValue)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return err
	}
	return nil
}

// Get attempts to retreive the value behind the supplied key.
// It returns a interface{} with either the retreived value or the default value and any error encountered.
// If supplied key is not found and defaultValue is set to nil it returns a KeyNotFoundError
// If supplied key path goes deeper into a non-map type (string, int, bool) it returns a UnexpectedValueTypeError
func (jconfig *JsonConfig) Get(key string, defaultValue interface{}) (interface{}, error) {
	parts := strings.Split(key, ".")
	var tmp interface{} = jconfig.obj
	for index, part := range parts {
		if len(part) == 0 {
			continue
		}
		if confMap, ok := tmp.(map[string]interface{}); ok {
			if value, exists := confMap[part]; exists {
				tmp = value
			} else if defaultValue != nil {
				return defaultValue, nil
			} else {
				return nil, &KeyNotFoundError{key: path.Join(append(parts[:index], part)...)}
			}
		} else if confMap, ok := tmp.(map[interface{}]interface{}); ok {
			if value, exists := confMap[part]; exists {
				tmp = value
			} else if defaultValue != nil {
				return defaultValue, nil
			} else {
				return nil, &KeyNotFoundError{key: path.Join(append(parts[:index], part)...)}
			}
		} else {
			return nil, &UnexpectedValueTypeError{key: path.Join(parts[:index]...), value: tmp, message: "value behind key is not a map[string]interface{}"}
		}
	}
	return tmp, nil
}
