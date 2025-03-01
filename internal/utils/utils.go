package utils

import (
	"fmt"
	"os"
	"strconv"
)

func LookupEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if ok {
		return value, nil
	}
	return value, nil
}

func GetBool(key string, def bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("failed to parste %s: %v", key, err)
	}

	return b, nil
}
