package genplugin

import "testing"

func TestUniqueTypeName_collisions(t *testing.T) {
	used := map[string]int{}
	if got := UniqueTypeName("catalog.Id", used); got != "Id" {
		t.Fatalf("first = %q", got)
	}
	if got := UniqueTypeName("orders.Id", used); got != "orders_Id" {
		t.Fatalf("second = %q", got)
	}
	if got := UniqueTypeName("orders.Other.Id", used); got != "orders_Other_Id" {
		t.Fatalf("third same short name = %q", got)
	}
}
