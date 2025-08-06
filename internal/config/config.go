package config

import (
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Auth   `yaml:"auth"`
	Server `yaml:"server"`
}

func LoadConfig() (c Config, err error) {
	f, err := os.Open("config.yaml")
	if err != nil {
		return
	}
	content, err := io.ReadAll(f)
	if err != nil {
		return
	}
	defer f.Close()

	err = yaml.Unmarshal(content, &c)
	if err != nil {
		return
	}

	return
}
