package transformerts

import (
	"reflect"
	"testing"
)

func TestSortDedupeStrings(t *testing.T) {
	got := sortDedupeStrings([]string{"b", "a", "", "b", "a"})
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
