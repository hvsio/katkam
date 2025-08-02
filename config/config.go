package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	AuthConfig `yaml:"auth"`
	Server     `yaml:"server"`
}

func LoadConfig() (c Config, err error) {

	f, err := os.Open("config.yaml")
	if err != nil {
		return
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&c)
	if err != nil {
		return
	}

	return
}
