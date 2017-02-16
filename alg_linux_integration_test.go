//+build linux

package alg_test

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"testing"

	"github.com/mdlayher/alg"
)

const MB = (1 << 20)

var buf = bytes.Repeat([]byte("a"), 512*MB)

// Flags to specify using either stdlib or AF_ALG transformations.
var (
	flagBenchSTD = flag.Bool("bench.std", false, "benchmark only standard library transformations")
	flagBenchALG = flag.Bool("bench.alg", false, "benchmark only AF_ALG transformations")
)

func init() {
	flag.Parse()
}

func TestMD5Equal(t *testing.T) {
	const expect = "0829f71740aab1ab98b33eae21dee122"
	withHash(t, "md5", func(algh hash.Hash) {
		testHashEqual(t, expect, md5.New(), algh)
	})
}

func TestSHA1Equal(t *testing.T) {
	const expect = "0631457264ff7f8d5fb1edc2c0211992a67c73e6"
	withHash(t, "sha1", func(algh hash.Hash) {
		testHashEqual(t, expect, sha1.New(), algh)
	})
}

func TestSHA256Equal(t *testing.T) {
	const expect = "9f1dcbc35c350d6027f98be0f5c8b43b42ca52b7604459c0c42be3aa88913d47"
	withHash(t, "sha256", func(algh hash.Hash) {
		testHashEqual(t, expect, sha256.New(), algh)
	})
}

func BenchmarkMD5(b *testing.B) {
	withHash(b, "md5", func(algh hash.Hash) {
		benchmarkHashes(b, md5.New(), algh)
	})
}

func BenchmarkSHA1(b *testing.B) {
	withHash(b, "sha1", func(algh hash.Hash) {
		benchmarkHashes(b, sha1.New(), algh)
	})
}

func BenchmarkSHA256(b *testing.B) {
	withHash(b, "sha256", func(algh hash.Hash) {
		benchmarkHashes(b, sha256.New(), algh)
	})
}

func limitReader(size int64) io.Reader {
	return io.LimitReader(bytes.NewBuffer(buf), size)
}

func testHashEqual(t *testing.T, expect string, stdh, algh hash.Hash) {
	const n = 8192

	w := io.MultiWriter(stdh, algh)
	r := limitReader(n)

	if nn, err := io.Copy(w, r); err != nil || int64(nn) != n {
		t.Fatalf("failed to copy: %q\n- want bytes: %d\n-  got bytes: %d",
			err, n, nn)
	}

	cb := stdh.Sum(nil)
	ab := algh.Sum(nil)
	log.Printf("%x\n%x", cb, ab)

	if want, got := cb, ab; !bytes.Equal(want, got) {
		t.Fatalf("unexpected hash sum:\n- std: %x\n- alg: %x", want, got)
	}

	if want, got := expect, hex.EncodeToString(ab); want != got {
		t.Fatalf("unexpected golden hash:\n- want: %q\n-  got: %q",
			want, got)
	}
}

func benchmarkHashes(b *testing.B, stdh, algh hash.Hash) {
	sizes := []int64{
		/*
			1,
			32,
			64,
			128,
			256,
		*/
		512,
	}

	pages := []int{
		16,
		64,
		128,
		256,
		512,
	}

	for _, size := range sizes {
		for _, page := range pages {
			name := fmt.Sprintf("%dMB/%dpages", size, page)
			switch {
			case *flagBenchSTD && *flagBenchALG:
				b.Fatal("cannot specify both '-bench.std' and '-bench.alg'")
			case *flagBenchSTD:
				b.Run(name, func(b *testing.B) {
					benchmarkHash(b, size*MB, page, stdh)
				})
			case *flagBenchALG:
				b.Run(name, func(b *testing.B) {
					benchmarkHash(b, size*MB, page, algh)
				})
			default:
				b.Run(name+"/std", func(b *testing.B) {
					benchmarkHash(b, size*MB, page, stdh)
				})

				b.Run(name+"/alg", func(b *testing.B) {
					benchmarkHash(b, size*MB, page, algh)
				})
			}
		}
	}
}

func benchmarkHash(b *testing.B, n int64, pages int, h hash.Hash) {
	copyBuf := make([]byte, os.Getpagesize()*pages)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r := limitReader(n)

		if nn, err := io.CopyBuffer(h, r, copyBuf); err != nil || int64(nn) != n {
			b.Fatalf("failed to copy: %q\n- want bytes: %d\n-  got bytes: %d",
				err, n, nn)
		}

		h.Sum(nil)
		h.Reset()
	}
}

func withHash(tb testing.TB, name string, fn func(h hash.Hash)) {
	c, err := alg.Dial("hash", name, nil)
	if err != nil {
		tb.Fatalf("failed to dial kernel: %v", err)
	}
	defer c.Close()

	h, err := c.Hash()
	if err != nil {
		tb.Fatalf("failed to make hash: %v", err)
	}
	defer h.Close()

	fn(h)
}
