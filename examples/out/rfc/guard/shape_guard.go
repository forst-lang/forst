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

// T_488eVThFocF: TypeDefShapeExpr({input: {name: String}, ctx: AppContext})
type T_488eVThFocF struct {
	ctx   AppContext `json:"ctx"`
	input User       `json:"input"`
}

// T_8XMnUftkRoa: TypeDefShapeExpr({sessionId: Pointer(String), user: Pointer(User)})
type T_8XMnUftkRoa struct {
	sessionId *string `json:"sessionId"`
	user      *User   `json:"user"`
}

// T_F1jpghi8Uyp: TypeDefShapeExpr({input: {name: String}})
type T_F1jpghi8Uyp struct {
	input User `json:"input"`
}

// T_LYQafBLM8TQ: TypeDefShapeExpr({sessionId: Value(Ref(Variable(sessionId))), user: {name: Value("Alice")}})
type T_LYQafBLM8TQ struct {
	sessionId string `json:"sessionId"`
	user      User   `json:"user"`
}

// T_Yo2G7tbwv5q: TypeDefShapeExpr({ctx: {sessionId: Value(Ref(Variable(sessionId))), user: {name: Value("Alice")}}, input: {name: Value("Fix memory leak in Node.js app")}})
type T_Yo2G7tbwv5q struct {
	ctx   T_LYQafBLM8TQ `json:"ctx"`
	input User          `json:"input"`
}

// T_b7WdLNpLZoy: TypeDefShapeExpr({ctx: {sessionId: Value(nil)}, input: {name: Value("Go to the gym")}})
type T_b7WdLNpLZoy struct {
	ctx   *string `json:"ctx"`
	input User    `json:"input"`
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
	name, err := createTask(T_488eVThFocF{ctx: AppContext{sessionId: &sessionId, user: &User{name: "Alice"}}, input: User{name: "Fix memory leak in Node.js app"}})
	if err != nil {
		println(err.Error())
		panic(errors.New("assertion failed: " + "Error.Nil()"))
	}
	println("Correctly created task: " + name)
	name, err = createTask(T_488eVThFocF{ctx: AppContext{sessionId: nil}, input: User{name: "Go to the gym"}})
	if err == nil {
		println("Expected error but gym task was created")
		panic(errors.New("assertion failed: " + "Error.Present()"))
	}
	println("Correctly avoided creating gym task as user was not logged in")
}
