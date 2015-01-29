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
	conf, err := globalconf.New("alfred")
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		os.Exit(1)
	}

	// Initialize configuration, reading from environment variables using a 'ALFRED_' prefix first,
	// then moving to a static configuration file, usually located in ~/.config/alfred/config.ini.
	conf.EnvPrefix = "ALFRED_"
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
