package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type LocalDriveConfig struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Config struct {
	Listen string             `json:"listen"`
	Locals []LocalDriveConfig `json:"locals"`
}

func LoadConfig(file string) (*Config, error) {
	configFile, e := os.Open(file)
	if e != nil {
		return nil, e
	}
	defer func() { _ = configFile.Close() }()
	bytes, e := ioutil.ReadAll(configFile)
	if e != nil {
		return nil, e
	}
	config := new(Config)
	e = json.Unmarshal(bytes, config)
	if e != nil {
		return nil, e
	}
	return config, nil
}
