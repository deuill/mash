package service

import (
	// Standard library
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	// Third-party packages
	"github.com/julienschmidt/httprouter"
	"github.com/rakyll/globalconf"
)

var (
	port     *string            // The port number on which the internal HTTP service will listen.
	services map[string]bool    // A map of services indexed under their name.
	router   *httprouter.Router // The default router for all incoming requests.
)

// A HandleFunc represents the default signature for registered methods attached to services.
type HandleFunc func(http.ResponseWriter, *http.Request, Params) (interface{}, error)

// Handler represents a registered handler method attached to Alfred.
type Handler struct {
	Method string     // The HTTP method handler is attached under, e.g. GET, POST, DELETE etc.
	Path   string     // The request path to bind handler against. Supports parameter bindings.
	Handle HandleFunc // The method to use for this handler.
}

// Params are attached to methods according to their path declarations, and may contain values
// corresponding to named parameters declared on those paths.
type Params httprouter.Params

// Get returns the value corresponding to a named parameter. It returns an empty string if no value
// was found.
func (p Params) Get(name string) string {
	return httprouter.Params(p).ByName(name)
}

// Register service for use with Alfred.
func Register(name string, flags *flag.FlagSet, handlers []Handler) error {
	if _, exists := services[name]; exists {
		return fmt.Errorf("Service '%s' already exists, refusing to overwrite", name)
	}

	services[name] = true

	if flags != nil {
		globalconf.Register(name, flags)
	}

	for _, h := range handlers {
		hv := h.Handle
		handle := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			if result, err := hv(w, r, Params(p)); err != nil {
				writeResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			} else if result != nil {
				writeResponse(w, http.StatusOK, result)
			}
		}

		path := "/" + name + h.Path
		router.Handle(h.Method, path, handle)
	}

	return nil
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

// Initialize services for Alfred, including internal HTTP service.
func Init() error {
	var err error

	// Start HTTP server, sending any errors back to the 'result' channel.
	result := make(chan error)
	go func() {
		result <- http.ListenAndServe(":"+*port, router)
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

// Initialize internal resources and configuration variables.
func init() {
	router = httprouter.New()
	services = make(map[string]bool)

	// Define configuration variables used for the HTTP service.
	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	port = fs.String("port", "6116", "")

	globalconf.Register("http", fs)
}
