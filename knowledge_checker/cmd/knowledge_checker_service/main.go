package main

import (
	"flag"
	"os"

	appication "github.com/parta4ok/kvs/knowledge_checker/pkg/application"
)

func main() {
	var configPath string

	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("KVS_CONFIG_PATH")
	}

	if configPath == "" {
		panic("config path is not set")
	}

	app := &appication.App{
		CfgPath: configPath,
	}

	app.Start()
}
