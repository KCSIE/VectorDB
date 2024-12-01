package model

import (
	"fmt"
	"reflect"
	"strings"
)

type HNSWParams struct {
	EfConstruction int
	MMax           int
	Heuristic      bool
	Extend         bool
	MaxSize        int
}

type FlatParams struct {
	MaxSize int
}

type SearchResult struct {
	ID    string
	Score float32
}

func ValidateAndConvert(indexType string, params map[string]interface{}) (interface{}, error) {
	var result interface{}

	switch indexType {
	case "flat":
		result = &FlatParams{}
	case "hnsw":
		result = &HNSWParams{}
	default:
		return nil, fmt.Errorf("unsupported index type: %s", indexType)
	}

	val := reflect.ValueOf(result).Elem()
	for key, value := range params {
		field := val.FieldByNameFunc(func(s string) bool {
			return strings.EqualFold(s, key)
		})

		if !field.IsValid() {
			return nil, fmt.Errorf("invalid parameter key: %s", key)
		}

		fieldValue := reflect.ValueOf(value)
		if field.Kind() == reflect.Int {
			field.SetInt(int64(fieldValue.Float()))
		} else if field.Type() != fieldValue.Type() {
			return nil, fmt.Errorf("invalid type for key '%s', expected: %s but got: %s", key, field.Type(), fieldValue.Type())
		} else {
			field.Set(fieldValue)
		}
	}

	return result, nil
}
