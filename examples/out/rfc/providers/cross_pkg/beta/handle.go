package beta

import "providers_cross_pkg_demo/alpha"
// Logger: TypeDefShapeExpr({Info: ?})
type Logger interface {
	Info(msg string)
}
// NopLogger: TypeDefShapeExpr({})
type NopLogger struct {
}
type Providers_2TAwF8pWZKc struct {
	Logger alpha.Logger
}

func Handle(providers Providers_2TAwF8pWZKc, id string) {
	alpha.LogExpiry(alpha.Providers_2TAwF8pWZKc{Logger: providers.Logger}, id)
}
