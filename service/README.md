# Services in Alfred

By default, Alfred provides no built-in tasks, and instead acts as host to services, self-contained
pieces of code that attach themselves to Alfred according to a minimal set of rules.

## Structure

A service consists of a folder containing any number of `go` files in a package named after the
folder in which the files are contained. The package defines a custom type, which is registered with
Alfred, and methods attached to that type which correspond to available tasks within that service.

## Sample Service

In order to demonstrate the rules required for writing a service, we are going to build a sample
service named `helloworld`, which defines two tasks, one which responds with "Hello World!" and
another which responds with "Goodbye World!". The service also defines a configuration variable
`name` which overrides the default value of "World" and allows the service to respond with a
custom name (see.g. "Hello Alex!").

```go
package helloworld

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/Hearst-Digital/alfred/service"
)

type Helloworld struct {
	Name *string
}

func (h *Helloworld) Hello(r *http.Request, w http.ResponseWriter) (interface{}, error) {
	if *h.Name == "" {
		return "", fmt.Errorf("Name is empty!")
	}

	return "Hello " + *h.Name + "!", nil
}

func (h *Helloworld) Goodbye(r *http.Request, w http.ResponseWriter) (interface{}, error) {
	if *h.Name == "" {
		return "", fmt.Errorf("Name is empty!")
	}

	return "Goodbye " + *h.Name + "!", nil
}

func (h *Helloworld) Start() error {
	service.RegisterHandler("helloworld", "hello", h.Hello)
	service.RegisterHandler("helloworld", "goodbye", h.Goodbye)

	return nil
}

func (h *Helloworld) Stop() error {
	return nil
}

func init() {
	fs := flag.NewFlagSet("helloworld", flag.ContinueOnError)
	serv := &Helloworld{
		Name: fs.String("name", "World", ""),
	}

	service.Register("helloworld", serv, fs)
}
```

Services contain a minimal amount of boilerplate in order to be functional. These are as follows:

```go
type ServiceName interface{}
```

The method receiver (in this case, the `Helloworld` struct) can be any valid receiver type, including
an empty structure. The receiver is defined in the service host as:

```go
type Service interface {
	Start() error
	Stop() error
}
```

And any valid method receiver has to implement these two methods _at least_.

```go
func (s *ServiceName) Start() error
func (s *ServiceName) Stop() error
```

These two methods implement the `Service` interface defined in the service host. Method `Start`
executes once when Alfred starts up, before the internal HTTP service is initialized, and most
commonly contains calls to `service.RegisterHandler()`, used for attaching specific methods to
the service host (which then attaches them as URL endpoints, as explained below).

```go
func init()
```

This method contains, minimally, a call to `service.Register()` for attaching the method receiver
to the service host. Although technically possible, it is not recommended to attach more than one
method receiver per package.

Any command-line options are also declared here, and become available under the global configuration
scheme.

## Handling requests

After all registered services complete their initialization routine, the service host initializes its
internal HTTP server and begins accepting requests on a specified TCP port (default is 6116).

Methods registered using `service.RegisterHandler()` are made available under the URL scheme of
`/<servicename>/<methodname>/`, where `servicename` and `methodname` correspond to the first and
second parameters accepted by `service.RegisterHandler()`. Any method registered in this way is
expected to correspond to the following declaration:

```go
func (h *ServiceName) MethodName(r *http.Request, w http.ResponseWriter) (interface{}, error)
```

Methods are expected to handle any arguments bound to the HTTP request on their own via the
`http.Request` structure. However, a method returning a value or error will have those types
returned as JSON documents to the caller. If more control over the response is needed, the
`http.ResponseWriter` type is also available.

Use of this type and returning of values to be serialized is mutually exclusive, so choose whichever
fits best for you and return a `nil` interface{} type if a method is to use the ResponseWriter directly.