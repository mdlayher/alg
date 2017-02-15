//+build !linux

package alg

import (
	"testing"
)

func TestOthersConnUnimplemented(t *testing.T) {
	c := &conn{}
	want := errUnimplemented

	if _, got := dial("", "", nil); want != got {
		t.Fatalf("unexpected error during dial:\n- want: %v\n-  got: %v",
			want, got)
	}

	if _, got := c.Hash(0, 0); want != got {
		t.Fatalf("unexpected error during c.Hash:\n- want: %v\n-  got: %v",
			want, got)
	}

	if got := c.Close(); want != got {
		t.Fatalf("unexpected error during c.Close:\n- want: %v\n-  got: %v",
			want, got)
	}
}
