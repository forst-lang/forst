package main

import errors "errors"

// AppContext: TypeDefShapeExpr({sessionId: Pointer(String)})
type AppContext struct {
	sessionId *string
}

// AppMutation: TypeDefShapeExpr({ctx: AppContext})
type AppMutation struct {
	ctx AppContext
}

// MutationArg: TypeDefAssertionExpr(Shape)
type MutationArg struct {
}

// T_488eVThFocF: TypeDefShapeExpr({ctx: AppContext, input: {name: String}})
type T_488eVThFocF struct {
	ctx   AppContext
	input T_azh9nsqmxaF
}

// T_7zywpPhwVhj: TypeDefShapeExpr({name: Value("Alice")})
type T_7zywpPhwVhj struct {
	name string
}

// T_D4c1JHuBYcw: TypeDefShapeExpr({sessionId: Value(Ref(Variable(sessionId)))})
type T_D4c1JHuBYcw struct {
	sessionId *string
}

// T_EMV7npYWLDn: TypeDefAssertionExpr(Value("Alice"))
type T_EMV7npYWLDn string

// T_F1jpghi8Uyp: TypeDefShapeExpr({input: {name: String}})
type T_F1jpghi8Uyp struct {
	input T_azh9nsqmxaF
}

// T_S16voJ6uyzy: TypeDefShapeExpr({input: {name: Value("Alice")}, ctx: {sessionId: Value(Ref(Variable(sessionId)))}})
type T_S16voJ6uyzy struct {
	ctx   T_D4c1JHuBYcw
	input T_7zywpPhwVhj
}

// T_azh9nsqmxaF: TypeDefShapeExpr({name: String})
type T_azh9nsqmxaF struct {
	name string
}

// T_jhkroyier29: TypeDefAssertionExpr(Value(Ref(Variable(sessionId))))
type T_jhkroyier29 string

func G_DeDHUAU73V6(ctx AppContext) bool {
	if ctx.sessionId == nil {
		return false
	}
	return true
}

func createUser(op T_488eVThFocF) (string, error) {
	if !G_DeDHUAU73V6(op.ctx) {
		println("Not logged in")
		return "", errors.New("assertion failed: " + "LoggedIn()")
	}
	println(*op.ctx.sessionId)
	return op.input.name, nil
}

func main() {
	sessionId := "479569ae-cbf0-471e-b849-38a698e0cb69"
	createUser(T_488eVThFocF{ctx: AppContext{sessionId: &sessionId}, input: T_azh9nsqmxaF{name: "Alice"}})
}
