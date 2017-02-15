//+build !linux

package alg

import (
	"fmt"
	"runtime"
)

var (
	// errUnimplemented is returned by all functions on platforms that
	// cannot make use of AF_ALG sockets.
	errUnimplemented = fmt.Errorf("alg: AF_ALG sockets not implemented on %s/%s",
		runtime.GOOS, runtime.GOARCH)
)

// A conn is the no-op implementation of an AF_ALG socket connection.
type conn struct{}

// dial is the entry point for Dial.  dial always returns an error.
func dial(typ, name string, config *Config) (*conn, error) {
	return nil, errUnimplemented
}

// Close always returns an error.
func (c *conn) Close() error {
	return errUnimplemented
}

// Hash always returns an error.
func (c *conn) Hash(size, blockSize int) (Hash, error) {
	return nil, errUnimplemented
}
