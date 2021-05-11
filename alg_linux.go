//+build linux

package alg

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

const defaultSocketBufferSize = 64 * 1024

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
	FD() int
	Read(p []byte) (int, error)
	Sendto(p []byte, flags int, to unix.Sockaddr) error
}

// dial is the entry point for Dial. dial opens an AF_ALG socket
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

	pipes, err := newPipe()
	if err != nil {
		return nil, err
	}

	return &ihash{
		s:         s,
		buf:       make([]byte, 128),
		addr:      c.addr,
		size:      size,
		blockSize: blockSize,
		pipes:     pipes,
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

	pipes [2]pipe
}

// Close closes the ihash's socket.
func (h *ihash) Close() error {
	return h.s.Close()
}

func (h *ihash) ReadFrom(r io.Reader) (int64, error) {
	if f, ok := r.(*os.File); ok {
		if w, err, handled := h.sendfile(f, -1); handled {
			return w, err
		}
		if w, err, handled := h.splice(f, -1); handled {
			return w, err
		}
	}
	if lr, ok := r.(*io.LimitedReader); ok {
		return h.readFromLimitedReader(lr)
	}
	return genericReadFrom(h, r)
}

func (h *ihash) readFromLimitedReader(lr *io.LimitedReader) (int64, error) {
	if f, ok := lr.R.(*os.File); ok {
		if w, err, handled := h.sendfile(f, lr.N); handled {
			return w, err
		}
		if w, err, handled := h.splice(f, lr.N); handled {
			return w, err
		}
	}
	return genericReadFrom(h, lr)
}

func (h *ihash) splice(f *os.File, remain int64) (written int64, err error, handled bool) {
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, nil, false
	}
	fi, err := f.Stat()
	if err != nil {
		return 0, nil, false
	}
	if remain == -1 {
		remain = fi.Size() - offset
	}
	// mmap must align on a page boundary
	// mmap from 0, use data from offset
	mmap, err := syscall.Mmap(int(f.Fd()), 0, int(fi.Size()),
		syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return 0, nil, false
	}
	defer syscall.Munmap(mmap)
	bytes := mmap[offset : offset+remain]
	var (
		total = len(bytes)
		start = 0
		end   = defaultSocketBufferSize
	)

	if end > total {
		end = total
	}
	for {
		n, err := h.Write(bytes[start:end])
		if err != nil {
			return int64(start + n), err, true
		}
		start += n
		if start >= total {
			break
		}
		end += n
		if end > total {
			end = total
		}
	}
	return remain, nil, true
}

func (h *ihash) sendfile(f *os.File, remain int64) (written int64, err error, handled bool) {
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, nil, false
	}
	fi, err := f.Stat()
	if err != nil {
		return 0, nil, false
	}
	if remain == -1 {
		remain = fi.Size() - offset
	}
	sc, err := f.SyscallConn()
	if err != nil {
		return 0, nil, false
	}
	var (
		n    int
		werr error
	)
	err = sc.Read(func(fd uintptr) bool {
		for {
			n, werr = syscall.Sendfile(h.s.FD(), int(fd), &offset, int(remain))
			written += int64(n)
			if werr != nil {
				break
			}
			if int64(n) >= remain {
				break
			}
			remain -= int64(n)
		}
		return true
	})
	if err == nil {
		err = werr
	}
	return written, err, true
}

// Write writes data to an AF_ALG socket, but instructs the kernel
// not to finalize the hash.
func (h *ihash) Write(b []byte) (int, error) {
	n, err := h.pipes[1].Vmsplice(b, 0)
	if err != nil {
		return n, err
	}
	_, err = h.pipes[0].Splice(h.s.FD(), n, unix.SPLICE_F_MOVE|unix.SPLICE_F_MORE)
	return n, err
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
func (s *sysSocket) FD() int                     { return s.fd }
func (s *sysSocket) Sendto(p []byte, flags int, to unix.Sockaddr) error {
	return unix.Sendto(s.fd, p, flags, to)
}
func (s *sysSocket) Read(p []byte) (int, error) { return unix.Read(s.fd, p) }

func newPipe() ([2]pipe, error) {
	var pipes [2]int
	if err := unix.Pipe(pipes[:]); err != nil {
		return [2]pipe{}, err
	}

	return [2]pipe{
		&sysPipe{fd: pipes[0]},
		&sysPipe{fd: pipes[1]},
	}, nil
}

type pipe interface {
	Splice(out, size, flags int) (int64, error)
	Vmsplice(b []byte, flags int) (int, error)
}

type sysPipe struct {
	fd int
}

func (p *sysPipe) Splice(out, size, flags int) (int64, error) {
	return unix.Splice(p.fd, nil, out, nil, size, flags)
}

func (p *sysPipe) Vmsplice(b []byte, flags int) (int, error) {
	iov := unix.Iovec{
		Base: &b[0],
	}
	iov.SetLen(len(b))

	return unix.Vmsplice(
		p.fd,
		[]unix.Iovec{iov},
		flags,
	)
}

type writerOnly struct {
	io.Writer
}

// Fallback implementation of io.ReaderFrom's ReadFrom, when os.File isn't
// applicable.
func genericReadFrom(w io.Writer, r io.Reader) (n int64, err error) {
	// Use wrapper to hide existing r.ReadFrom from io.Copy.
	return io.Copy(writerOnly{w}, r)
}
