package main

import errors "errors"

// AppContext: TypeDefShapeExpr({sessionId: Pointer(String), user: Pointer(User)})
type AppContext struct {
	sessionId *string `json:"sessionId"`
	user      *User   `json:"user"`
}

// AppMutation: TypeDefShapeExpr({ctx: AppContext})
type AppMutation struct {
	ctx AppContext `json:"ctx"`
}

// MutationArg: TypeDefAssertionExpr(Shape)
type MutationArg struct {
}

// TYPE_POINTER: TypeDefAssertionExpr(Value(Ref(Variable(sessionId))))
type TYPE_POINTER string

// TYPE_STRING: TypeDefAssertionExpr(Value("Alice"))
type TYPE_STRING string

// T_488eVThFocF: TypeDefShapeExpr({ctx: AppContext, input: {name: String}})
type T_488eVThFocF struct {
	ctx   AppContext    `json:"ctx"`
	input T_azh9nsqmxaF `json:"input"`
}

// T_7zywpPhwVhj: TypeDefShapeExpr({name: Value("Alice")})
type T_7zywpPhwVhj struct {
	name string `json:"name"`
}

// T_DX1exfNyf6L: TypeDefShapeExpr({name: Value("Fix memory leak in Node.js app")})
type T_DX1exfNyf6L struct {
	name string `json:"name"`
}

// T_F1jpghi8Uyp: TypeDefShapeExpr({input: {name: String}})
type T_F1jpghi8Uyp struct {
	input User `json:"input"`
}

// T_LYQafBLM8TQ: TypeDefShapeExpr({sessionId: Value(Ref(Variable(sessionId))), user: {name: Value("Alice")}})
type T_LYQafBLM8TQ struct {
	sessionId *string `json:"sessionId"`
	user      User    `json:"user"`
}

// T_X86jJwVQ4mH: TypeDefShapeExpr({sessionId: Value(nil)})
type T_X86jJwVQ4mH struct {
	sessionId string `json:"sessionId"`
}

// T_Yo2G7tbwv5q: TypeDefShapeExpr({ctx: {sessionId: Value(Ref(Variable(sessionId))), user: {name: Value("Alice")}}, input: {name: Value("Fix memory leak in Node.js app")}})
type T_Yo2G7tbwv5q struct {
	ctx   AppContext `json:"ctx"`
	input User       `json:"input"`
}

// T_azh9nsqmxaF: TypeDefShapeExpr({name: String})
type T_azh9nsqmxaF struct {
	name string `json:"name"`
}

// T_b7WdLNpLZoy: TypeDefShapeExpr({ctx: {sessionId: Value(nil)}, input: {name: Value("Go to the gym")}})
type T_b7WdLNpLZoy struct {
	ctx   T_X86jJwVQ4mH `json:"ctx"`
	input T_7zywpPhwVhj `json:"input"`
}

// T_goedQ5siUEo: TypeDefShapeExpr({name: Value("Go to the gym")})
type T_goedQ5siUEo struct {
	name string `json:"name"`
}

// User: TypeDefShapeExpr({name: String})
type User struct {
	name string `json:"name"`
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
func createTask(op T_488eVThFocF) (string, error) {
	if !G_W5HnAr5D9X1(op.ctx) {
		return "", errors.New("assertion failed: " + "AppContext.LoggedIn()")
	}
	println("Creating task, logged in with sessionId: " + *op.ctx.sessionId)
	return op.input.name, nil
}
func main() {
	sessionId := "479569ae-cbf0-471e-b849-38a698e0cb69"
	name, err := createTask(T_488eVThFocF{ctx: AppContext{sessionId: &sessionId, user: &User{name: "Alice"}}, input: T_azh9nsqmxaF{name: "Fix memory leak in Node.js app"}})
	if err != nil {
		println(err.Error())
		panic(errors.New("assertion failed: " + "Error.Nil()"))
	}
	println("Correctly created task: " + name)
	name, err = createTask(T_488eVThFocF{ctx: AppContext{sessionId: nil}, input: T_azh9nsqmxaF{name: "Go to the gym"}})
	if err == nil {
		println("Expected error but gym task was created")
		panic(errors.New("assertion failed: " + "Error.Present()"))
	}
	println("Correctly avoided creating gym task as user was not logged in")
}
