package jsonfast

import (
	json "github.com/goccy/go-json"
)

func Valid(data []byte) bool {
	return json.Valid(data)
}

func Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func Unmarshal(data []byte, value any) error {
	return json.Unmarshal(data, value)
}
