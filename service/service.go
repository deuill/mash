package service

import (
	// Standard library
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	// Third-party packages
	"github.com/rakyll/globalconf"
)

// Service represents a canonical service implementation. Services are expected to handle request
// parameters on their own, and can optionally write to the request.
type Service interface {
	// Call calls named method on service, passing the raw HTTP request for processing. The method
	// can optionally write a response directly and return `nil` for both return parameters, or,
	// if the default JSON encoding is desired, return any type for marshalling.
	Call(method string, r *http.Request, w http.ResponseWriter) (interface{}, error)

	// Start initializes service.
	Start() error

	// Stop shuts the service down, performing any cleanup required.
	Stop() error
}

// The port number on which the internal HTTP service will listen.
var port *string

// A map of services attached to Alfred. Each service defines its own API, and may be introspectable
// using the OPTIONS method, either globally, or on a method in that service.
var services map[string]Service

// Register service for use with Alfred.
//
// Services are attached to Alfred under the name passed. Attaching multiple services under the same
// name is prohibited, and will result in an error being thrown.
//
// You may also pass a flag.Flagset for options registered globally, in which case they will become
// available to the environment according to the naming conventions used by each configuration method.
func Register(name string, srv Service, fs *flag.FlagSet) error {
	if _, exists := services[name]; exists {
		return fmt.Errorf("Service '%s' already exists, refusing to overwrite", name)
	}

	if fs != nil {
		globalconf.Register(name, fs)
	}

	services[name] = srv

	return nil
}

// Initialize services for Alfred, including internal HTTP service.
func Init() error {
	var err error

	// Start HTTP server, sending any errors back to the 'result' channel.
	result := make(chan error)
	go func() {
		http.HandleFunc("/", handleRequest)
		result <- http.ListenAndServe(":"+*port, nil)
	}()

	// Allow for a 500 millisecond timeout before this function returns, in order to catch any
	// errors that might be emitted by the HTTP server goroutine.
	timeout := make(chan bool)
	go func() {
		time.Sleep(500 * time.Millisecond)
		timeout <- true
	}()

	// Start services registered on Alfred.
	for name, s := range services {
		if err = s.Start(); err != nil {
			return fmt.Errorf("[%s]: %s", name, err)
		}
	}

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

// Route HTTP request through to registered module, if any matches.
//
// A valid request consists of two parts, corresponding to the service and the method name to call.
// So, for a request path like: `/user/get`, this would (attempt to) call into Service `User`, method
// `Get`.
//
// Responses are encoded in JSON. If an error occurs at any stage of the request, the response will
// contain an 'error' field containing an error message, and an HTTP response code != 200.
func handleRequest(w http.ResponseWriter, r *http.Request) {
	split := strings.SplitN(strings.Trim(r.URL.Path, "/"), "/", 2)

	if len(split) < 2 {
		writeResponse(w, http.StatusBadRequest, map[string]string{
			"error": "Not enough arguments: service or method name is empty",
		})

		return
	}

	serv, method := split[0], split[1]
	result, err := services[serv].Call(method, r, w)

	// Check for error message, if any.
	if err != nil {
		writeResponse(w, http.StatusBadRequest, map[string]string{
			"error": serv + ": " + err.Error(),
		})

		return
	}

	// Only attempt to write a response if method returned a value to write.
	if result != nil {
		writeResponse(w, http.StatusOK, result)
	}
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

// Initialize internal resources and configuration variables.
func init() {
	services = make(map[string]Service)

	// Define configuration variables used for the HTTP service.
	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	port = fs.String("port", "6116", "")

	globalconf.Register("http", fs)
}
