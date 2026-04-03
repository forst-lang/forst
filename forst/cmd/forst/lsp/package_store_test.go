package lsp

import (
	"testing"
)

func TestPackageStore_RegisterFile_SamePathReturnsStableID(t *testing.T) {
	t.Parallel()
	ps := NewPackageStore()
	id1 := ps.RegisterFile("/project/src/app.ft", "project/src")
	id2 := ps.RegisterFile("/project/src/app.ft", "project/src")
	if id1 != id2 {
		t.Fatalf("ids differ: %q vs %q", id1, id2)
	}
	byPath, ok := ps.GetFileByPath("/project/src/app.ft")
	if !ok || byPath != id1 {
		t.Fatalf("GetFileByPath = %q ok=%v", byPath, ok)
	}
}

func TestPackageStore_GetFileInfo(t *testing.T) {
	t.Parallel()
	ps := NewPackageStore()
	id := ps.RegisterFile("/x/y/foo.ft", "x/y")
	fi, ok := ps.GetFileInfo(id)
	if !ok || fi.Filename != "foo.ft" || fi.Path != "/x/y/foo.ft" {
		t.Fatalf("FileInfo = %#v ok=%v", fi, ok)
	}
}

func TestPackageStore_GetFileMetadata(t *testing.T) {
	t.Parallel()
	ps := NewPackageStore()
	id := ps.RegisterFile("/abs/doc.ft", "abs")
	meta := ps.GetFileMetadata(id)
	if meta == nil {
		t.Fatal("nil metadata")
	}
	if meta.URI != "file:///abs/doc.ft" || meta.Filename != "doc.ft" {
		t.Fatalf("%+v", meta)
	}
}
