package genplugin

import "testing"

func TestRoutePath_remixStyle(t *testing.T) {
	got, err := RoutePath("app/api/orders/$id.ft", "app/api", "$id")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/api/orders/:id" {
		t.Fatalf("got %q", got)
	}
}

func TestRoutePath_nextStyle(t *testing.T) {
	got, err := RoutePath("app/api/orders/[id].ft", "app/api", "[id]")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/api/orders/:id" {
		t.Fatalf("got %q", got)
	}
}

func TestRoutePath_static(t *testing.T) {
	got, err := RoutePath("app/api/orders/place.ft", "app/api", "$id")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/api/orders/place" {
		t.Fatalf("got %q", got)
	}
}

func TestRRPath_stripsLeadingSlash(t *testing.T) {
	got, err := RRPath("app/api/orders/$id.ft", "app/api", "$id")
	if err != nil {
		t.Fatal(err)
	}
	if got != "api/orders/:id" {
		t.Fatalf("got %q", got)
	}
}

func TestRoutePath_rejectsOutsideRoot(t *testing.T) {
	_, err := RoutePath("other/routes.ft", "app/api", "$id")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHandlerStem(t *testing.T) {
	if got := HandlerStem("app/api/orders/$id.ft", "app/api"); got != "orders.$id" {
		t.Fatalf("got %q", got)
	}
}

func TestPathParamNames(t *testing.T) {
	got := PathParamNames("app/api/orders/$id.ft")
	if len(got) != 1 || got[0] != "id" {
		t.Fatalf("got %#v", got)
	}
}
