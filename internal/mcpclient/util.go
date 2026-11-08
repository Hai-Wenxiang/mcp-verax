package mcpclient

import (
	"bufio"
	"io"
)

// newLineScanner builds a scanner with a generous buffer for server stderr.
func newLineScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 8<<20)
	return sc
}
