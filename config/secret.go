package config

import (
	"io/ioutil"
	"strings"
)

func LoadSecret(filename string) (string, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf)), nil
}
