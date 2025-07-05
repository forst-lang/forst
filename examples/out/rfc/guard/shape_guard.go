package main

import errors "errors"

// AppContext: TypeDefShapeExpr({sessionId: Pointer(String), user: Pointer(User)})
type AppContext struct {
	sessionId *string
	user      *User
}

// AppMutation: TypeDefShapeExpr({ctx: AppContext.})
type AppMutation struct {
	ctx AppContext
}

// MutationArg: TypeDefAssertionExpr(Shape.)
type MutationArg struct {
}

// T_488eVThFocF: TypeDefShapeExpr({ctx: AppContext., input: {name: String}})
type T_488eVThFocF struct {
	ctx   AppContext
	input T_azh9nsqmxaF
}

// T_4jjqhYo1BLN: TypeDefAssertionExpr(Value("Charlie"))
type T_4jjqhYo1BLN string

// T_5cA53Q8Mmbo: TypeDefAssertionExpr(Value("Bob"))
type T_5cA53Q8Mmbo string

// T_76s5BSD9SUz: TypeDefShapeExpr({name: Value("Charlie")})
type T_76s5BSD9SUz struct {
	name string
}

// T_7zywpPhwVhj: TypeDefShapeExpr({name: Value("Alice")})
type T_7zywpPhwVhj struct {
	name string
}

// T_8fV3pHKb2Vw: TypeDefShapeExpr({ctx: {sessionId: Value(nil)}, input: {name: Value("Charlie")}})
type T_8fV3pHKb2Vw struct {
	ctx   T_X86jJwVQ4mH
	input T_76s5BSD9SUz
}

// T_AkitGn3xqxS: TypeDefShapeExpr({ctx: {sessionId: Value(Ref(Variable(sessionId))), user: {name: Value("Alice")}}, input: {name: Value("Bob")}})
type T_AkitGn3xqxS struct {
	ctx   T_LYQafBLM8TQ
	input T_E86CFGh7pHJ
}

// T_E86CFGh7pHJ: TypeDefShapeExpr({name: Value("Bob")})
type T_E86CFGh7pHJ struct {
	name string
}

// T_EMV7npYWLDn: TypeDefAssertionExpr(Value("Alice"))
type T_EMV7npYWLDn string

// T_F1jpghi8Uyp: TypeDefShapeExpr({input: {name: String}})
type T_F1jpghi8Uyp struct {
	input T_azh9nsqmxaF
}

// T_LYQafBLM8TQ: TypeDefShapeExpr({user: {name: Value("Alice")}, sessionId: Value(Ref(Variable(sessionId)))})
type T_LYQafBLM8TQ struct {
	sessionId *string
	user      T_7zywpPhwVhj
}

// T_X86jJwVQ4mH: TypeDefShapeExpr({sessionId: Value(nil)})
type T_X86jJwVQ4mH struct {
	sessionId string
}

// T_azh9nsqmxaF: TypeDefShapeExpr({name: String})
type T_azh9nsqmxaF struct {
	name string
}

// T_bN37LzcUYJ8: TypeDefAssertionExpr(Value(nil))
type T_bN37LzcUYJ8 string

// T_jhkroyier29: TypeDefAssertionExpr(Value(Ref(Variable(sessionId))))
type T_jhkroyier29 string

// User: TypeDefShapeExpr({name: String})
type User struct {
	name string
}

func G_W5HnAr5D9X1(ctx AppContext) bool {
	if ctx.sessionId == nil {
		return false
	}
	if ctx.user == nil {
		return false
	}
	return true
}

func createUser(op T_488eVThFocF) (string, error) {
	if !G_W5HnAr5D9X1(op.ctx) {
		return "", errors.New("assertion failed: " + "AppContext.LoggedIn()")
	}
	println("Creating user, logged in with sessionId: " + *op.ctx.sessionId)
	return op.input.name, nil
}

func main() {
	sessionId := "479569ae-cbf0-471e-b849-38a698e0cb69"
	name, err := createUser(T_488eVThFocF{ctx: AppContext{sessionId: &sessionId, user: &User{name: "Alice"}}, input: T_azh9nsqmxaF{name: "Bob"}})
	if err != nil {
		println(err.Error())
		panic(errors.New("assertion failed: " + "Error.Nil()"))
	}
	println("Created user: " + name)
	name, err = createUser(T_488eVThFocF{ctx: AppContext{sessionId: nil}, input: T_azh9nsqmxaF{name: "Charlie"}})
	if err == nil {
		println("Expected error but user Charlie was created")
		panic(errors.New("assertion failed: " + "Error.Present()"))
	}
	println("Correctly avoided creating user Charlie")
}
