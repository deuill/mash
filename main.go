package main

import (
	// Standard library
	"fmt"
	"os"
	"os/signal"

	// Internal packages
	"github.com/Hearst-Digital/alfred/service"

	// Third-party packages
	"github.com/rakyll/globalconf"
)

// Entry point for Alfred, this sets up global configuration and starts internal services.
func main() {
	// Allow one to override the default configuration file location using the ALFRED_CONFIG env
	// variable. By definition, this variable exists outside of the configuration file and as such
	// doesn't follow the same semantics as other configuration variables.
	configFile := os.Getenv("ALFRED_CONFIG")
	if configFile == "" {
		configFile = "/etc/alfred/alfred.conf"
	}

	// Initialize configuration, reading from environment variables using a 'ALFRED_' prefix first,
	// then moving to a static configuration file, usually located in '/etc/alfred/alfred.conf'.
	conf, err := globalconf.NewWithOptions(&globalconf.Options{configFile, "ALFRED_"})
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

	// Listen for and terminate Alfred on SIGKILL or SIGINT signals.
	sigStop := make(chan os.Signal)
	signal.Notify(sigStop, os.Interrupt, os.Kill)

	select {
	case <-sigStop:
		fmt.Println("Shutting down server...")

		errs := service.Shutdown()
		if errs != nil {
			fmt.Println("The following services failed to shut down cleanly:")
			for _, err = range errs {
				fmt.Println(err)
			}

			fmt.Println("The environment might be in an unclean state")
			os.Exit(2)
		}
	}
}
