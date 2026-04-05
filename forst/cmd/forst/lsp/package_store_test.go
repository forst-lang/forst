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
	if meta.URI != fileURIForLocalPath("/abs/doc.ft") || meta.Filename != "doc.ft" {
		t.Fatalf("%+v", meta)
	}
}

func TestPackageStore_GetFileMetadata_unknownIDReturnsNil(t *testing.T) {
	t.Parallel()
	ps := NewPackageStore()
	if m := ps.GetFileMetadata(FileID("missing")); m != nil {
		t.Fatalf("got %+v", m)
	}
}

func TestPackageStore_GetPackageInfo(t *testing.T) {
	t.Parallel()
	ps := NewPackageStore()
	id := ps.RegisterFile("/proj/a/foo.ft", "proj/a")
	pkg, ok := ps.GetPackageInfo(PackageID("proj/a"))
	if !ok || pkg == nil {
		t.Fatalf("GetPackageInfo ok=%v pkg=%v", ok, pkg)
	}
	if pkg.Name != "a" || pkg.Path != "proj/a" {
		t.Fatalf("pkg = %#v", pkg)
	}
	if len(pkg.Files) != 1 || pkg.Files[id] == nil {
		t.Fatalf("files = %d", len(pkg.Files))
	}
}

func TestPackageStore_twoFilesSamePackage(t *testing.T) {
	t.Parallel()
	ps := NewPackageStore()
	id1 := ps.RegisterFile("/proj/p/x.ft", "proj/p")
	id2 := ps.RegisterFile("/proj/p/y.ft", "proj/p")
	pkg, ok := ps.GetPackageInfo(PackageID("proj/p"))
	if !ok || len(pkg.Files) != 2 {
		t.Fatalf("want 2 files in package, got ok=%v len=%d", ok, len(pkg.Files))
	}
	if pkg.Files[id1] == nil || pkg.Files[id2] == nil {
		t.Fatal("missing file entries")
	}
	all := ps.GetAllFiles()
	if len(all) != 2 {
		t.Fatalf("GetAllFiles len = %d", len(all))
	}
	allPkgs := ps.GetAllPackages()
	if len(allPkgs) != 1 {
		t.Fatalf("GetAllPackages len = %d", len(allPkgs))
	}
}
