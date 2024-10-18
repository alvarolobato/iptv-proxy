package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"
)

// ErrorWithLocation wraps an error with file and line information
func ErrorWithLocation(err error) error {
	if err == nil {
		return nil
	}

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("error occurred: %v", err)
	}

	return fmt.Errorf("%s:%d: %v", file, line, err)
}

// PrintJSONErrorContext returns a string containing the context around a JSON error
func PrintJSONErrorContext(data []byte, offset int64) string {
	start := offset - 20
	if start < 0 {
		start = 0
	}
	end := offset + 20
	if end > int64(len(data)) {
		end = int64(len(data))
	}
	return strings.Replace(string(data[start:end]), "\n", " ", -1)
}

func UnmarshalReflectiveFields(data []byte, v interface{}, fieldName string) error {
	var objMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &objMap); err != nil {
		return fmt.Errorf("error unmarshaling %s: %v", fieldName, err)
	}

	valuePtr := reflect.ValueOf(v)
	if valuePtr.Kind() != reflect.Ptr {
		return fmt.Errorf("%s must be a pointer", fieldName)
	}
	value := valuePtr.Elem()

	// Create a map to track which fields have been processed
	processedFields := make(map[string]bool)

	// Create a slice to store errors
	var errors []string

	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = field.Name
		}
		if jsonTag == "-" {
			continue
		}
		jsonTag = strings.Split(jsonTag, ",")[0]

		processedFields[jsonTag] = true

		if rawValue, ok := objMap[jsonTag]; ok {
			// Check if the value is empty or an empty array
			if len(rawValue) == 0 || string(rawValue) == "\"\"" || string(rawValue) == "[]" || string(rawValue) == "[null]" {
				continue
			}

			fieldValue := value.Field(i)
			if fieldValue.CanSet() {
				err := json.Unmarshal(rawValue, fieldValue.Addr().Interface())
				if err != nil {
					errMsg := fmt.Sprintf("Error unmarshaling field %s.%s (value: %s): %v", fieldName, field.Name, string(rawValue), err)
					log.Printf("Warning: %s", errMsg)
					errors = append(errors, errMsg)
					// Continue with other fields instead of returning an error
				}
			}
		}
	}

	/*
	   // Log fields in the JSON that are not in the struct
	   for jsonField, rawValue := range objMap {
	       if !processedFields[jsonField] {
	           var value interface{}
	           err := json.Unmarshal(rawValue, &value)
	           if err != nil {
	               log.Printf("Warning: Error unmarshaling extra field %s.%s: %v", fieldName, jsonField, err)
	               // } else {
	               //  log.Printf("Extra field in %s: %s = %v", fieldName, jsonField, value)
	           }
	       }
	   }
	*/

	// If there were any errors during the process, return an error
	if len(errors) > 0 {
		return fmt.Errorf("unmarshalReflectiveFields encountered %d error(s) for %s: %s", len(errors), fieldName, strings.Join(errors, "; "))
	}

	return nil
}
