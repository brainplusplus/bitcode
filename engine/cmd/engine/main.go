package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
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

	serverErr := make(chan error, 1)
	go func() {
		if err := app.Start(); err != nil {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		fmt.Println("Shutting down...")
	case err := <-serverErr:
		if isServerClosed(err) {
			return
		}
		log.Fatalf("server error: %v", err)
	}

	if err := app.Shutdown(); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func isServerClosed(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "server closed") || strings.Contains(msg, "use of closed network connection")
}
