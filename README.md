# The Go Programming Language

Go is an open source programming language that makes it easy to build simple,
reliable, and efficient software.

![Gopher image](https://golang.org/doc/gopher/fiveyears.jpg)
*Gopher image by [Renee French][rf], licensed under [Creative Commons 3.0 Attributions license][cc3-by].*

This repository is a *downstream fork* of https://go.googlesource.com/go

It is intended to support ports that are not (yet?) accepted upstream,
or ports which have been demoted from upstream.

While we do not require that a port pass all tests, there are some rules:

* go-ports is a downstream fork, and superset of, the latest release
* a go-ports port can be promoted to upstream, if it makes sense
* an upstream port can be demoted from upstream but remain in go-ports, assuming a maintainer
* anything on the main go-ports branch has to build
* there is no requirement that all ports pass all tests, though this is encouraged
* ports that do not "just build" will be placed on a branch
* a port unique to go-ports must have a maintainer listed in the MAINTAINERS file
* ports that lose maintainers will be placed on a branch

Unless otherwise noted, the Go source files are distributed under the
BSD-style license found in the LICENSE file.

### Download and Install

### Contributing

Go is the work of thousands of contributors. We appreciate your help!

go-ports uses standard github PR workflows; in this it differs from upstream.
That said, to ensure easy code movement, we require that you sign the Go CLA.

[rf]: https://reneefrench.blogspot.com/
[cc3-by]: https://creativecommons.org/licenses/by/3.0/
