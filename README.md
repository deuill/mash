# Mash - the simple task runner

[![MIT License][license-svg]][license-url]

Mash is a simple task runner -- a service that listens on a standard TCP port and accepts commands
over a RESTful interface. It is written in Go and is designed to be simple to build, deploy and use.

## Building

Assuming you have all already installed build dependancies required via `go get`, building Mash is
simply a matter of running `make` in the project root. You may install Mash by running `make install`
or build a redistributable package using `make package`.

## Running

You may either run Mash directly using the 'mash' binary, or use the supplied init script, which
will also handle permissions and locking. By default, Mash listens on port `6116` and does not need
elevated permissions for operation.

## Configuration

Mash requires a minimal set of initial configuration, and relies on a good set of defaults for most
operations. It can either use environment variables, or a local configuration file located in
`/etc/mash/mash.conf` (the default location can be changed by setting an `MASH_CONFIG`
environment variable) for overriding default values, using the following semantics:

Configuration values are namespaced under their service name and option key. Environment variables
use an `MASH_` prefix, and are uppercase, while `mash.conf` variables are placed in sections
using the service name as a key, and are lowercase. So, for an option `port` under service `http`,
the following methods could be used to set the corresponding variable to `8080`:

```shell
export MASH_HTTP_PORT=8080
```

set in the environment in which `mash` is launched, or:

```ini
[http]
port = 8080
```

set as a persistent value in the local file. Environment variables override file variables, which in
turn override defaults.

## License

Mash is licensed under the MIT license, the terms of which can be found in the included LICENSE file.

[license-url]: https://github.com/deuill/mash/blob/master/LICENSE
[license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
