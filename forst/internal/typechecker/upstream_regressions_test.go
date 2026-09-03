package typechecker

import (
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func typecheckMustOK(t *testing.T, src string) {
	t.Helper()
	dir := moduleRootFromWD(t)
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
}

func TestRegression_cryptoSHA256AndHMAC(t *testing.T) {
	t.Parallel()
	typecheckMustOK(t, `package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func FingerprintTag(pepper String, value String): String {
	h := sha256.New()
	h.Write([]byte(pepper + "|" + value))
	sum := h.Sum([]byte{})
	return hex.EncodeToString(sum)
}

func useHMAC(key String, msg String): Int {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(msg))
	out := mac.Sum([]byte{})
	return len(out)
}

func main() {
	println(FingerprintTag("p", "v"))
	println(useHMAC("k", "m"))
}
`)
}

func TestRegression_goStmt(t *testing.T) {
	t.Parallel()
	typecheckMustOK(t, `package main

import "fmt"

func printHi() {
	fmt.Println("hi")
}

func main() {
	go printHi()
}
`)
}

func TestRegression_newGoNamedType(t *testing.T) {
	t.Parallel()
	typecheckMustOK(t, `package main

import "net/http"

func main() {
	r := new(http.Request)
	println(r != nil)
}
`)
}

func TestRegression_goPtrCompositeLit(t *testing.T) {
	t.Parallel()
	typecheckMustOK(t, `package main

import "net/url"

func main() {
	u := &url.URL{ Path: "/x" }
	println(u.Path)
}
`)
}

func TestRegression_httpErrServerClosedEq(t *testing.T) {
	t.Parallel()
	typecheckMustOK(t, `package main

import "net/http"

func check(err Error) {
	if err == http.ErrServerClosed {
		return
	}
}

func main() {
	check(http.ErrServerClosed)
}
`)
}
