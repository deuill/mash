package service

import (
	// Standard library
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	_ "net/http/pprof"

	// Third-party packages
	"github.com/rakyll/globalconf"
)

type Service interface {
	Start() error // Start initializes service.
	Stop() error  // Stop shuts the service down, performing any cleanup required.
}

// A map of services indexed under their name.
var services map[string]Service

// A handler represents the default signature for registered methods attached to services.
type handler func(r *http.Request, w http.ResponseWriter) (interface{}, error)

// Register service for use with Alfred. Services are attached to Alfred under the name passed.
// Attaching multiple services under the same name is prohibited, and will result in an error being
// thrown.
//
// You may also pass a flag.Flagset for options registered globally, in which case they will
// become available to the environment according to the naming conventions used by each configuration
// method.
func Register(name string, serv Service, fs *flag.FlagSet) error {
	if _, exists := services[name]; exists {
		return fmt.Errorf("Service '%s' already exists, refusing to overwrite", name)
	}

	if fs != nil {
		globalconf.Register(name, fs)
	}

	services[name] = serv

	return nil
}

// RegisterHandler attaches handler methods owned by services to specific URL patterns.
func RegisterHandler(serv, method string, h handler) {
	http.HandleFunc("/"+serv+"/"+method+"/", func(w http.ResponseWriter, r *http.Request) {
		result, err := h(r, w)
		if err != nil {
			writeResponse(w, http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})

			return
		}

		// Only attempt to write a response if method returned a value to write.
		if result != nil {
			writeResponse(w, http.StatusOK, result)
		}
	})
}

// Encode response in JSON and write to connection.
func writeResponse(w http.ResponseWriter, code int, data interface{}) {
	// All responses are sent in UTF8-encoded JSON.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	b, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(w, "{error: \"%s\"}", err)
		return
	}

	w.Write(b)
	return
}

// The port number on which the internal HTTP service will listen.
var port *string

// Initialize services for Alfred, including internal HTTP service.
func Init() error {
	var err error

	// Start services registered on Alfred.
	for name, s := range services {
		if err = s.Start(); err != nil {
			return fmt.Errorf("[%s]: %s", name, err)
		}
	}

	// Start HTTP server, sending any errors back to the 'result' channel.
	result := make(chan error)
	go func() {
		result <- http.ListenAndServe(":"+*port, nil)
	}()

	// Allow for a 500 millisecond timeout before this function returns, in order to catch any
	// errors that might be emitted by the HTTP server goroutine.
	timeout := make(chan bool)
	go func() {
		time.Sleep(500 * time.Millisecond)
		timeout <- true
	}()

	select {
	// The HTTP server has returned before the timeout, which means that an error has occured.
	case err = <-result:
		return err
	// A timeout has occurred and no error has been received by the server, which probably
	// means that everything went OK.
	case <-timeout:
		return nil
	}
}

// Shut down all registered services, by calling each service's Stop method. Collects and returns
// a slice of errors for each Stop method returning one, or nil if no errors occured.
func Shutdown() []error {
	var err error
	var errors []error

	for name, s := range services {
		if err = s.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("[%s]: %s", name, err))
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Initialize internal resources and configuration variables.
func init() {
	services = make(map[string]Service)

	// Define configuration variables used for the HTTP service.
	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	port = fs.String("port", "6116", "")

	globalconf.Register("http", fs)
}
