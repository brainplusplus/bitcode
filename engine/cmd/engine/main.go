package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitcode-engine/engine/internal"
)

func main() {
	configFile := flag.String("config", "", "Path to bitcode.yaml config file")
	flag.Parse()

	configPath := *configFile
	if configPath == "" {
		configPath = os.Getenv("CONFIG_FILE")
	}

	cfg, err := internal.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app, err := internal.NewApp(cfg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	if err := app.LoadModules(); err != nil {
		log.Fatalf("failed to load modules: %v", err)
	}

	go func() {
		if err := app.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
	if err := app.Shutdown(); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
