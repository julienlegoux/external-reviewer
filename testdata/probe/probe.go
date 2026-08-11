// Package probe exercises every write-API pattern manually. It is a scratch
// file used to verify .golangci.yml's forbidigo coverage; run it by hand:
//
//	golangci-lint run ./testdata/probe/...
package probe

import (
	"os"
	"time"
)

func probe() {
	_ = os.Chmod("x", 0)
	_ = os.Chtimes("x", time.Time{}, time.Time{})
	_ = os.Truncate("x", 0)
	_ = os.Link("a", "b")

	root, _ := os.OpenRoot("x")
	_ = root.Chmod("x", 0)
	_ = root.Link("a", "b")

	f, _ := os.Open("x")
	_, _ = f.Write(nil)
	_, _ = f.WriteString("x")
	_ = f.Truncate(0)
}
