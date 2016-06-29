# Services in Mash

By default, Mash provides no built-in tasks, and instead acts as host to services, self-contained pieces of code that attach themselves to Mash according to a minimal set of rules.

## Structure

A service consists of a folder containing any number of `go` files in a package named after the folder in which the files are contained. The package defines a custom type, which is registered with Mash, and methods attached to that type which correspond to available tasks within that service.

## Sample Service

In order to demonstrate the rules required for writing a service, we are going to build a sample service named `helloworld`, which defines two tasks, one which responds with "Hello World!" and another which responds with "Goodbye World!".

The service also defines a configuration variable `name` which overrides the default value of "World" and allows the service to respond with a custom name (e.g. "Hello Alex!").

```go
package helloworld

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/deuill/mash/service"
)

type Helloworld struct {
	Name *string
}

func (h *Helloworld) Hello(w http.ResponseWriter, r *http.Request, p service.Params) (interface{}, error) {
	if *h.Name == "" {
		return "", fmt.Errorf("Name is empty!")
	}

	return "Hello " + *h.Name + "!", nil
}

func (h *Helloworld) Goodbye(w http.ResponseWriter, r *http.Request, p service.Params) (interface{}, error) {
	if *h.Name == "" {
		return "", fmt.Errorf("Name is empty!")
	}

	return "Goodbye " + *h.Name + "!", nil
}

func init() {
	flags := flag.NewFlagSet("helloworld", flag.ContinueOnError)
	serv := &Helloworld{
		Name: flags.String("name", "World", ""),
	}

	service.Register("helloworld", flags, []service.Handler{
		{"GET", "/hello", serv.Hello},
		{"GET", "/goodbye", serv.Goodbye},
	})
}
```

Services contain a minimal amount of boilerplate in order to be functional. These are as follows:

```go
type ServiceName interface{}
```

The method receiver (in this case, the `Helloworld` struct) can be any valid receiver type, including an empty structure. This receiver is then used to register any attached methods to the service host, as explained below.

```go
func init()
```

This method contains, minimally, a call to `service.Register()` for attaching the method receiver and any exported methods to the service host. Although technically possible, it is not recommended to attach more than one method receiver per package.

Any command-line options are also declared here, and become available under the global configuration scheme.

## Handling requests

After all registered services complete their initialization routine, the service host initializes its internal HTTP server and begins accepting requests on a specified TCP port (default is `6116`).

Methods registered using `service.Register` are made available under their service name, followed by the path specified in their Handler type. Paths are matched according to rules specified in the [httprouter Documentation](https://github.com/julienschmidt/httprouter).

So, for the example calls above, you would get the following URL endpoints, for a local server running with the default options:

```
http://localhost:6116/helloworld/hello
http://localhost:6116/helloworld/goodbye
```

Any method registered in this way is expected to correspond to the following declaration:

```go
func (*ServiceName) MethodName(http.ResponseWriter, *http.Request, service.Params) (interface{}, error)
```

Methods can handle any arguments bound to the HTTP request via the `service.Params` type, which allows you to fetch named parameters via the `Params.Get` method, or on their own using the `http.Request` type.

Returning data to the user can be accomplished by returning any non-`nil` data, in which case the values are encoded as JSON before being returned, or manually through the `http.ResponseWriter` type, in which case the method is expected to return `nil` for the `interface{}` type.
