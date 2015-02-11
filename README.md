# Alfred - the simple task runner

Alfred is a simple task runner -- a service that listens on a standard TCP port and accepts commands
over a RESTful interface. It is written in Go and is designed to be simple to build, deploy and use.

## Building

Assuming you have all already installed build dependancies required via `go get`, building Alfred is
simply a matter of running `make` in the project root. You may install Alfred by running `make install`
or build a redistributable package using `make package`.

Alternatively, you can build by running `go build` in the project root.

## Running

You may either run Alfred directly using the 'alfred' binary, or use the supplied init script, which
will also handle permissions and locking. By default, Alfred listens on port `6116` and does not need
elevated permissions for operation.

## Configuration

Alfred requires a minimal set of initial configuration, and relies on a good set of defaults for most
operations. However, it can either use environment variables, or a local configuration file located in
`/etc/alfred/alfred.conf` (the default location can be changed by setting an `ALFRED_CONFIG` environment
variable) for overriding default values, using the following semantics:

Configuration values are namespaced under their service name and option key. Environment variables
use an 'ALFRED_' prefix, and are uppercase, while `alfred.conf` variables are placed in sections
using the service name as a key, and are lowercase. So, for an option 'port' under service 'http',
the following methods could be used to set the corresponding variable to `8080`:

```shell
export ALFRED_HTTP_PORT=8080
```

set in the environment in which `alfred` is launched, or:

```ini
[http]
port = 8080
```

set as a persistent value in the local file. Environment variables override file variables, which in
turn override defaults.