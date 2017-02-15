package alg_test

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"

	"github.com/mdlayher/alg"
)

func ExampleConn_hashSHA1() {
	// Dial the kernel using AF_ALG sockets. The socket must be closed when it
	// is no longer needed.
	c, err := alg.Dial(alg.SHA1())
	if err != nil {
		log.Fatalf("failed to dial kernel: %v", err)
	}
	defer c.Close()

	// Retrieve a hash handle from the kernel. This can be used the same as any
	// other hash.Hash, but must also be closed when it is no longer needed.
	h, err := c.Hash()
	if err != nil {
		log.Fatalf("failed to create hash: %v", err)
	}
	defer h.Close()

	if _, err := io.WriteString(h, "hello, world"); err != nil {
		log.Fatalf("failed to hash string: %v", err)
	}

	fmt.Println(hex.EncodeToString(h.Sum(nil)))
}
