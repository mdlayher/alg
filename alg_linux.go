//+build linux

package alg

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

// A conn is the internal connection type for Linux.
type conn struct {
	s    socket
	addr *unix.SockaddrALG
}

// A socket is a wrapper around socket-related system calls, to enable
// easier testing.
type socket interface {
	Accept() (socket, error)
	Bind(sa unix.Sockaddr) error
	Close() error
	Read(p []byte) (int, error)
	Sendto(p []byte, flags int, to unix.Sockaddr) error
}

// dial is the entry point for Dial.  dial opens an AF_ALG socket
// using system calls.
func dial(typ, name string, config *Config) (*conn, error) {
	fd, err := unix.Socket(unix.AF_ALG, unix.SOCK_SEQPACKET, 0)
	if err != nil {
		return nil, err
	}

	addr := &unix.SockaddrALG{
		Type:    typ,
		Name:    name,
		Feature: config.Feature,
		Mask:    config.Mask,
	}

	return bind(&sysSocket{fd: fd}, addr)
}

// bind binds an AF_ALG socket using the input socket, which may be
// a system call implementation or a mocked one for tests.
func bind(s socket, addr *unix.SockaddrALG) (*conn, error) {
	if err := s.Bind(addr); err != nil {
		return nil, err
	}

	return &conn{
		s:    s,
		addr: addr,
	}, nil
}

// Close closes a conn's socket.
func (c *conn) Close() error {
	return c.s.Close()
}

// Hash creates a new Hash handle by accepting a single connection and
// setting up an ihash.
func (c *conn) Hash(size, blockSize int) (Hash, error) {
	s, err := c.s.Accept()
	if err != nil {
		return nil, err
	}

	return &ihash{
		s:         s,
		buf:       make([]byte, 128),
		addr:      c.addr,
		size:      size,
		blockSize: blockSize,
	}, nil
}

var _ Hash = &ihash{}

// An ihash is the internal Linux implementation of Hash.
type ihash struct {
	s         socket
	buf       []byte
	addr      *unix.SockaddrALG
	size      int
	blockSize int
}

// Close closes the ihash's socket.
func (h *ihash) Close() error {
	return h.s.Close()
}

// Write writes data to an AF_ALG socket, but instructs the kernel
// not to finalize the hash.
func (h *ihash) Write(b []byte) (int, error) {
	err := h.s.Sendto(b, unix.MSG_MORE, h.addr)
	return len(b), err
}

// Sum reads data from an AF_ALG socket, and appends it to the input
// buffer.
func (h *ihash) Sum(b []byte) []byte {
	n, err := h.s.Read(h.buf)
	if err != nil {
		panic(fmt.Sprintf("alg: failed to read out finalized hash: %v", err))
	}

	return append(b, h.buf[:n]...)
}

// Reset is a no-op for AF_ALG sockets.
func (h *ihash) Reset() {}

// BlockSize returns the block size of the hash.
func (h *ihash) BlockSize() int { return h.blockSize }

// Size returns the size of the hash.
func (h *ihash) Size() int { return h.size }

// A sysSocket is a socket which uses system calls for socket operations.
type sysSocket struct {
	fd int
}

func (s *sysSocket) Accept() (socket, error) {
	fd, _, errno := unix.Syscall(unix.SYS_ACCEPT, uintptr(s.fd), 0, 0)
	if errno != 0 {
		return nil, syscall.Errno(errno)
	}

	// A sysSocket produces more sysSockets.
	return &sysSocket{
		fd: int(fd),
	}, nil
}
func (s *sysSocket) Bind(sa unix.Sockaddr) error { return unix.Bind(s.fd, sa) }
func (s *sysSocket) Close() error                { return unix.Close(s.fd) }
func (s *sysSocket) Sendto(p []byte, flags int, to unix.Sockaddr) error {
	return unix.Sendto(s.fd, p, flags, to)
}
func (s *sysSocket) Read(p []byte) (int, error) { return unix.Read(s.fd, p) }
