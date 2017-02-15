// Package alg provides access to Linux AF_ALG sockets for communication
// with the Linux kernel crypto API.
package alg

import (
	"fmt"
	"hash"
	"io"
)

// A Conn is a connection to the Linux kernel crypto API, using an AF_ALG
// socket.  A Conn can be used to initialize Hashes via its Hash method,
// using the parameters configured in Dial.
type Conn struct {
	size      int
	blockSize int

	c *conn
}

// A Config contains optional parameters for a Conn.
type Config struct {
	Feature uint32
	Mask    uint32
}

// Dial dials a connection the Linux kernel crypto API, using the specified
// transformation type, algorithm name, and optional configuration.  If config
// is nil, a default configuration will be used.
//
// At this time, the following transformation types and algorithm types are
// supported:
//   - hash
//     - md5
//     - sha1
//     - sha256
func Dial(typ, name string, config *Config) (*Conn, error) {
	if config == nil {
		config = &Config{}
	}

	var size, blockSize int
	var ok bool

	switch typ {
	case typeHash:
		size, blockSize, ok = hashSizes(name)
		if !ok {
			return nil, fmt.Errorf("alg: unknown hash algorithm %q", name)
		}
	default:
		return nil, fmt.Errorf("alg: transformation type %q unsupported", typ)
	}

	// Internal, OS-specific constructor for a conn.
	c, err := dial(typ, name, config)
	if err != nil {
		return nil, err
	}

	return &Conn{
		size:      size,
		blockSize: blockSize,

		c: c,
	}, nil
}

// Hash creates a Hash handle from a Conn.  The handle is not safe for
// concurrent use.
func (c *Conn) Hash() (Hash, error) {
	return c.c.Hash(c.size, c.blockSize)
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.c.Close()
}

// A Hash is a hash.Hash, with an added Close method.  Use it just as you
// would with a normal hash.Hash.  The Hash's Close method must be called
// to release its resources when it is no longer needed.
type Hash interface {
	hash.Hash
	io.Closer
}

// MD5 is a convenience function for use in Dial, to open a Conn that produces
// MD5 Hashes.
func MD5() (string, string, *Config) {
	return typeHash, nameMD5, nil
}

// SHA1 is a convenience function for use in Dial, to open a Conn that produces
// SHA1 Hashes.
func SHA1() (string, string, *Config) {
	return typeHash, nameSHA1, nil
}

// SHA256 is a convenience function for use in Dial, to open a Conn that produces
// SHA256 Hashes.
func SHA256() (string, string, *Config) {
	return typeHash, nameSHA256, nil
}

const (
	// Transformation types.
	typeHash = "hash"

	// Algorithm names.
	nameMD5    = "md5"
	nameSHA1   = "sha1"
	nameSHA256 = "sha256"
)

// hashSizes looks up a hash by its name and returns its size and block
// size, if available.  If the hash is not found, false will be returned.
func hashSizes(name string) (size, blockSize int, ok bool) {
	switch name {
	case nameMD5:
		return 16, 64, true
	case nameSHA1:
		return 20, 64, true
	case nameSHA256:
		return 32, 64, true
	}

	return 0, 0, false
}
