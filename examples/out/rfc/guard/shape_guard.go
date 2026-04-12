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
// T_WLVwGpNp6sG: TypeDefShapeExpr({})
type T_WLVwGpNp6sG struct {
}
// T_bTwM5AcoxRu: TypeDefShapeExpr({})
type T_bTwM5AcoxRu struct {
}
// T_bWZeLfb2t2d: TypeDefShapeExpr({})
type T_bWZeLfb2t2d struct {
}
// User: TypeDefShapeExpr({name: String})
type User struct {
	name string `json:"name"`
}

func G_PP94eAdBHT9(ctx AppContext) bool {
	if ctx.sessionId == nil {
		return false
	}
	if ctx.user == nil {
		return false
	}
	return true
}
func createTask(op T_488eVThFocF) (string, error) {
	if !G_PP94eAdBHT9(op.ctx) {
		return "", errors.New("assertion failed: AppContext.LoggedIn()")
	}
	println("Creating task, logged in with sessionId: " + *op.ctx.sessionId)
	return op.input.name, nil
}
func main() {
	sessionId := "479569ae-cbf0-471e-b849-38a698e0cb69"
	createTask(T_488eVThFocF{ctx: AppContext{user: &User{}, sessionId: &sessionId}, input: User{name: "Fix memory leak in Node.js app"}})
	println("Correctly created task (Result API)")
	createTask(T_488eVThFocF{ctx: AppContext{sessionId: nil, user: nil}, input: User{name: "Go to the gym"}})
	println("Correctly avoided creating gym task as user was not logged in")
}
