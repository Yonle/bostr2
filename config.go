package main

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
)

type Config struct {
	Listen  string
	Relays  []string
	NIP_11  map[string]interface{} `yaml:"nip_11"`
	Favicon string
}

func ReadConfig(filename string, c *Config) {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprintf("error when reading %s: %s", filename, err))
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		panic(fmt.Sprintf("error when parsing %s: %s", filename, err))
	}
}
