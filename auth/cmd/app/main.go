package main

import (
	"flag"
	"os"

	appication "github.com/parta4ok/kvs/auth/pkg/application"
)

func main() {
	var configPath string

	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("AUTH_CONFIG_PATH")
	}

	if configPath == "" {
		panic("config path is not set")
	}

	app := &appication.App{
		CfgPath: configPath,
	}

	app.Start()
}
