package web

import (
	"testing"
)

func TestGetFS(t *testing.T) {
	fsys := GetFS()
	if fsys == nil {
		t.Fatalf("expected non-nil FileSystem")
	}
}
