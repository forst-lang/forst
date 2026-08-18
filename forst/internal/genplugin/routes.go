package genplugin

import (
	"fmt"
	"path/filepath"
	"strings"
)

func toSlash(p string) string {
	return filepath.ToSlash(strings.TrimSpace(p))
}

// RoutePath maps a module-relative .ft path under routesRoot to an HTTP path.
// Filenames `$id` / `[id]` become `:id`. The `app/` directory is not part of the URL
// (Remix / Next convention): `app/api/orders/$id.ft` → `/api/orders/:id`.
func RoutePath(spanFile, routesRoot, paramStyle string) (string, error) {
	rel, err := routeRelative(spanFile, routesRoot, paramStyle)
	if err != nil {
		return "", err
	}
	return "/" + rel, nil
}

// RRPath is RoutePath without the leading slash, for `route("api/orders/:id", ...)`.
func RRPath(spanFile, routesRoot, paramStyle string) (string, error) {
	return routeRelative(spanFile, routesRoot, paramStyle)
}

func routeRelative(spanFile, routesRoot, _ string) (string, error) {
	spanFile = toSlash(spanFile)
	routesRoot = strings.Trim(toSlash(routesRoot), "/")
	if spanFile == "" {
		return "", fmt.Errorf("empty span.file")
	}
	if routesRoot == "" {
		return "", fmt.Errorf("routesRoot is required")
	}
	prefix := routesRoot + "/"
	if !strings.HasPrefix(spanFile, prefix) && spanFile != routesRoot && spanFile != routesRoot+".ft" {
		return "", fmt.Errorf("file %q is not under routesRoot %q", spanFile, routesRoot)
	}
	stripped := spanFile
	if strings.HasPrefix(stripped, "app/") {
		stripped = strings.TrimPrefix(stripped, "app/")
	}
	stripped = strings.TrimSuffix(stripped, ".ft")
	if stripped == "" {
		return "", fmt.Errorf("file %q produced an empty URL path", spanFile)
	}
	segments := strings.Split(stripped, "/")
	for i, seg := range segments {
		segments[i] = paramToURL(seg)
	}
	return strings.Join(segments, "/"), nil
}

func paramToURL(seg string) string {
	if name, ok := paramName(seg); ok {
		return ":" + name
	}
	return seg
}
