package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"github.com/goccy/go-yaml"
	"log"
	"os"
	"strings"
)

type Config struct {
	Listen  string
	Relays  []string
	NIP_11  map[string]interface{} `yaml:"nip_11"`
	Favicon string
}

//go:embed config.example.yaml
var exampleConf []byte

func createConf(filename string) {
	fmt.Printf("Do you want to create config file? (%s) [y/N]: ", filename)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	v := scanner.Text()

	switch strings.ToLower(v) {
	case "y":
		if err := os.WriteFile(filename, exampleConf, 0644); err != nil {
			panic(fmt.Sprintf("failed to create config file: %s", err))
		}

		fmt.Printf("\nConfig file has been written to %s\n", filename)
		fmt.Println("Please edit the config before starting bouncer.")
		fmt.Println("\nAfter editing, Start the bouncer by running:")

		if filename == "config.yaml" {
			fmt.Println("  $ bostr2")
		} else {
			fmt.Printf("  $ bostr2 -configfile %s\n", filename)
		}

		os.Exit(0)
		return
	default:
		fmt.Println("bostr2 cannot run without config.")
		os.Exit(1)
		return
	}
}

func ReadConfig(filename string, c *Config) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Config file does not exist.")
			createConf(filename)
		}

		panic(fmt.Sprintf("error when reading %s: %s", filename, err))
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		panic(fmt.Sprintf("error when parsing %s: %s", filename, err))
	}
}
