package api

import (
	"encoding/json"
	"fmt"
)

type IOMap map[string]IOValue

type IOValue struct {
	IsInt    bool
	IntValue int
	MapValue IOMap
}

func NewIOMap() IOMap {
	return make(IOMap)
}

func (iv IOValue) MarshalJSON() ([]byte, error) {
	if iv.IsInt {
		return json.Marshal(iv.IntValue)
	}
	return json.Marshal(iv.MapValue)
}

func (iv *IOValue) UnmarshalJSON(data []byte) error {
	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		iv.IsInt = true
		iv.IntValue = intValue
		return nil
	}

	var mapValue map[string]IOValue
	if err := json.Unmarshal(data, &mapValue); err == nil {
		iv.IsInt = false
		iv.MapValue = mapValue
		return nil
	}

	return fmt.Errorf("cannot unmarshal value")
}

func (iomap IOMap) IsValid() bool {
	// Collect all integer values in the map
	intValues := make(map[int]bool)
	for _, value := range iomap {
		if value.IsInt {
			intValues[value.IntValue] = true
		} else {
			for _, v := range value.MapValue {
				intValues[v.IntValue] = true
			}
		}
	}
	// Check if the values start with 0 and form a consecutive sequence
	for i := 0; i < len(intValues); i++ {
		if !intValues[i] {
			return false
		}
	}
	// If all values form a consecutive sequence, return true
	return true
}

func (iomap IOMap) ToMapInterface() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range iomap {
		if value.IsInt {
			result[key] = value.IntValue
		} else {
			result[key] = value.MapValue
		}
	}
	return result
}

func (iomap IOMap) GetInputArray(body map[string]interface{}) []interface{} {
	data := map[int]interface{}{}
	for key, ioValue := range iomap {
		if ioValue.IsInt {
			index := ioValue.IntValue
			value := body[key]
			data[index] = value
		} else {
			populateDataRecursively(ioValue.MapValue, body[key].(map[string]interface{}), &data)
		}
	}
	result := make([]interface{}, len(data))
	for i, v := range data {
		result[i] = v
	}
	return result
}

func populateDataRecursively(inputMap map[string]IOValue, body map[string]interface{}, data *map[int]interface{}) {
	for mapKey, mapValue := range inputMap {
		if mapValue.IsInt {
			index := mapValue.IntValue
			(*data)[index] = body[mapKey]
		} else {
			if nestedMap, ok := body[mapKey].(map[string]interface{}); ok {
				populateDataRecursively(mapValue.MapValue, nestedMap, data)
			}
		}
	}
}

func (iomap IOMap) ToMapOutput(data []interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range iomap {
		if value.IsInt {
			index := value.IntValue
			if index >= 0 && index < len(data) {
				result[key] = data[index]
			}
		} else {
			if value.IsInt {
				mappedValues := make(map[string]interface{})
				for k, v := range value.MapValue {
					if v.IntValue >= 0 && v.IntValue < len(data) {
						mappedValues[k] = data[v.IntValue]
					}
				}
				result[key] = mappedValues
			} else {
				result[key] = value.MapValue.ToMapOutput(data)
			}
		}
	}
	return result
}
