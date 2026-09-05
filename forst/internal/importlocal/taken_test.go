package importlocal

import "testing"

func TestTakenSet(t *testing.T) {
	s := make(TakenSet)
	s.Add("fmt")
	s.Add("")
	if !s.Has("fmt") {
		t.Fatal("expected fmt taken")
	}
	if s.Has("os") {
		t.Fatal("os should not be taken")
	}
	clone := s.Clone()
	if !clone.Has("fmt") {
		t.Fatal("clone missing fmt")
	}
	clone.Add("os")
	if s.Has("os") {
		t.Fatal("original should not include os after clone mutation")
	}
	m := s.Map()
	if _, ok := m["fmt"]; !ok {
		t.Fatal("Map missing fmt")
	}
}
