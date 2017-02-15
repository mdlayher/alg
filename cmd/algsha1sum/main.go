// Command algsha1sum is a Go implementation of sha1sum that uses package alg
// to perform the SHA1 hashing operation.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mdlayher/alg"
)

func main() {
	flag.Parse()

	c, err := alg.Dial(alg.SHA1())
	if err != nil {
		log.Fatalf("failed to dial kernel: %v", err)
	}
	defer c.Close()

	h, err := c.Hash()
	if err != nil {
		log.Fatalf("failed to create hash: %v", err)
	}
	defer h.Close()

	var r io.Reader
	arg := flag.Arg(0)
	switch arg {
	case "", "-":
		arg = "-"
		r = os.Stdin
	default:
		f, err := os.Open(arg)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		r = f
	}

	if _, err := io.Copy(h, r); err != nil {
		log.Fatalf("failed to copy: %v", err)
	}

	fmt.Printf("%x  %s\n", h.Sum(nil), arg)
}
