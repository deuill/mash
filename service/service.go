package service

import (
	// Standard library
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"

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
		handle := h.Handle
		call := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			if result, err := handle(w, r, Params(p)); err != nil {
				respond(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			} else if result != nil {
				respond(w, http.StatusOK, result)
			}
		}

		path := "/" + name + h.Path
		router.Handle(h.Method, path, call)
	}

	return nil
}

// Encode response in JSON and write to connection.
func respond(w http.ResponseWriter, code int, data interface{}) {
	// All responses are sent in UTF8-encoded JSON.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	b, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(w, `{error: "%s"}`, err)
		return
	}

	w.Write(append(b, '\n'))
	return
}

// Initialize service host, including internal HTTP service.
func Init() error {
	ln, err := net.Listen("tcp", net.JoinHostPort("", *port))
	if err != nil {
		return err
	}

	go http.Serve(ln, router)

	return nil
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
