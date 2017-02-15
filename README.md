alg [![Build Status](https://travis-ci.org/mdlayher/alg.svg?branch=master)](https://travis-ci.org/mdlayher/alg) [![GoDoc](https://godoc.org/github.com/mdlayher/alg?status.svg)](https://godoc.org/github.com/mdlayher/alg) [![Go Report Card](https://goreportcard.com/badge/github.com/mdlayher/alg)](https://goreportcard.com/report/github.com/mdlayher/alg)
===

Package `alg` provides access to Linux `AF_ALG` sockets for communication
with the Linux kernel crypto API.  MIT Licensed.

This package should be considered experimental, and should almost certainly
not be used in place of Go's built-in cryptographic cipher and hash packages.

The benefit of `AF_ALG` sockets is that they enable access to the Linux kernel's
cryptography API, and may be able to use hardware acceleration to perform
certain transformations.  On systems with dedicated cryptography processing
hardware (or systems without assembly implementations of certain
transformations), using this package may result in a performance boost.

If this package does end up being useful for you, please do reach out!
I'd love to hear what you're doing with it.

Benchmarking
------------

To benchmark `AF_ALG` transformations vs. the Go standard library equivalents
on a given system, run the following commands:

```
$ go test -c
$ ./alg.test -bench.std -test.bench . | tee std.txt
$ ./alg.test -bench.alg -test.bench . | tee alg.txt
$ benchcmp std.txt alg.txt
```

The `benchcmp` utility can be installed using:

```
$ go get golang.org/x/tools/cmd/benchcmp
```
