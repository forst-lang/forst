package main

import (
	"errors"
	"fmt"
)

// AppContext: struct with sessionId
type T_AppContext struct {
	SessionId *string
}

// T_bzeMYgDed7q: shape with ctx: AppContext, input: struct{name string}
type T_bzeMYgDed7q struct {
	Ctx   T_AppContext
	Input struct {
		Name string
	}
}

// Guard: LoggedIn
func G_LoggedIn(ctx T_AppContext) bool {
	return ctx.SessionId != nil
}

// Guard: NotLoggedIn
func G_NotLoggedIn(ctx T_AppContext) bool {
	return ctx.SessionId == nil
}

// createUser function
func createUser(op T_bzeMYgDed7q) string {
	if !(G_LoggedIn(op.Ctx)) {
		println("Not logged in")
		panic(errors.New("assertion failed: LoggedIn()"))
	}
	fmt.Println(op.Ctx.SessionId)
	return op.Input.Name
}

func main() {
	// Example usage
	sessionId := "479569ae-cbf0-471e-b849-38a698e0cb69"
	op := T_bzeMYgDed7q{
		Ctx: T_AppContext{
			SessionId: &sessionId,
		},
		Input: struct{ Name string }{Name: "Alice"},
	}
	result := createUser(op)
	fmt.Println(result)
}
