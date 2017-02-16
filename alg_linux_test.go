//+build linux

package alg

import (
	"bytes"
	"reflect"
	"testing"

	"golang.org/x/sys/unix"
)

func TestLinuxConn_bind(t *testing.T) {
	addr := &unix.SockaddrALG{
		Type: "hash",
		Name: "sha1",
	}

	s := &testSocket{}
	if _, err := bind(s, addr); err != nil {
		t.Fatalf("failed to bind: %v", err)
	}

	if want, got := addr, s.bind; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected bind address:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestLinuxConnWrite(t *testing.T) {
	addr := &unix.SockaddrALG{
		Type: "hash",
		Name: "sha1",
	}

	h, s := testLinuxHash(t, addr)

	b := []byte("hello world")
	if _, err := h.Write(b); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	if want, got := b, s.sendto.p; !bytes.Equal(want, got) {
		t.Fatalf("unexpected sendto bytes:\n- want: %v\n-  got: %v",
			want, got)
	}

	if want, got := unix.MSG_MORE, s.sendto.flags; want != got {
		t.Fatalf("unexpected sendto flags:\n- want: %v\n-  got: %v",
			want, got)
	}

	if want, got := addr, s.sendto.to; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected sendto addr:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestLinuxConnSum(t *testing.T) {
	addr := &unix.SockaddrALG{
		Type: "hash",
		Name: "sha1",
	}

	h, s := testLinuxHash(t, addr)
	s.read = []byte("deadbeef")

	sum := h.Sum([]byte("foo"))

	if want, got := []byte("foodeadbeef"), sum; !bytes.Equal(want, got) {
		t.Fatalf("unexpected sum bytes:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func testLinuxHash(t *testing.T, addr *unix.SockaddrALG) (Hash, *testSocket) {
	s := &testSocket{}
	c, err := bind(s, addr)
	if err != nil {
		t.Fatalf("failed to bind: %v", err)
	}

	hash, err := c.Hash(0, 0)
	if err != nil {
		t.Fatalf("failed to create hash: %v", err)
	}

	// A little gross, but gets the job done.
	return hash, hash.(*ihash).s.(*testSocket)
}

var _ socket = &testSocket{}

type testSocket struct {
	bind   unix.Sockaddr
	sendto struct {
		p     []byte
		flags int
		to    unix.Sockaddr
	}
	read []byte

	noopSocket
}

func (s *testSocket) Accept() (socket, error) {
	return &testSocket{}, nil
}
func (s *testSocket) Bind(sa unix.Sockaddr) error {
	s.bind = sa
	return nil
}
func (s *testSocket) Read(p []byte) (int, error) {
	n := copy(p, s.read)
	return n, nil
}
func (s *testSocket) Sendto(p []byte, flags int, to unix.Sockaddr) error {
	s.sendto.p = p
	s.sendto.flags = flags
	s.sendto.to = to
	return nil
}

var _ socket = &noopSocket{}

type noopSocket struct{}

func (s *noopSocket) Accept() (socket, error)                            { return nil, nil }
func (s *noopSocket) Bind(sa unix.Sockaddr) error                        { return nil }
func (s *noopSocket) Close() error                                       { return nil }
func (s *noopSocket) FD() int                                            { return 0 }
func (s *noopSocket) Read(p []byte) (int, error)                         { return 0, nil }
func (s *noopSocket) Sendto(p []byte, flags int, to unix.Sockaddr) error { return nil }
