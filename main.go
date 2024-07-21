package main

import (
	"flag"
	"fmt"
)

func main() {
	fmt.Println("bostr2 - bostr next generation")

	flag.StringVar(&Config_Filename, "configfile", "config.yaml", "Path to YAML config file")
	flag.Parse()

	LoadConfig()
	LoadFavicon()
	Serve()
}
