package main

import (
	// Standard library
	"fmt"
	"os"
	"os/signal"

	// Internal packages
	"github.com/deuill/mash/service"

	// Third-party packages
	"github.com/rakyll/globalconf"
)

// Entry point for Mash, this sets up global configuration and starts internal services.
func main() {
	// Allow one to override the default configuration file location using the MASH_CONFIG env
	// variable. By definition, this variable exists outside of the configuration file and as such
	// doesn't follow the same semantics as other configuration variables.
	configFile := os.Getenv("MASH_CONFIG")
	if configFile == "" {
		configFile = "/etc/mash/mash.conf"
	}

	// Initialize configuration, reading from environment variables using a 'MASH_' prefix first,
	// then moving to a static configuration file, usually located in '/etc/mash/mash.conf'.
	conf, err := globalconf.NewWithOptions(&globalconf.Options{configFile, "MASH_"})
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		os.Exit(1)
	}

	conf.ParseAll()
	fmt.Print("Starting server... ")

	// Initialize HTTP and attached services.
	err = service.Init()
	if err != nil {
		fmt.Printf("error initializing services:\n%s\n", err)
		os.Exit(1)
	}

	fmt.Println("done.")

	// Listen for and terminate Mash on SIGKILL or SIGINT signals.
	sigStop := make(chan os.Signal)
	signal.Notify(sigStop, os.Interrupt, os.Kill)

	select {
	case <-sigStop:
		fmt.Println("Shutting down server...")
	}
}
