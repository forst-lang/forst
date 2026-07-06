package main

import errors "errors"
// AppContext: TypeDefShapeExpr({sessionId: Pointer(String), user: Pointer(User)})
type AppContext struct {
	sessionId *string
	user      *User
}
// AppMutation: TypeDefShapeExpr({ctx: AppContext})
type AppMutation struct {
	ctx AppContext
}
// MutationArg: TypeDefShapeExpr({})
type MutationArg struct {
}
// User: TypeDefShapeExpr({name: String})
type User struct {
	name string
}
// T_488eVThFocF: TypeDefShapeExpr({ctx: AppContext, input: {name: String}})
type T_488eVThFocF struct {
	ctx   AppContext
	input User
}
// T_8XMnUftkRoa: TypeDefShapeExpr({sessionId: Pointer(String), user: Pointer(User)})
type T_8XMnUftkRoa struct {
	sessionId *string
	user      *User
}
// T_F1jpghi8Uyp: TypeDefShapeExpr({input: {name: String}})
type T_F1jpghi8Uyp struct {
	input User
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
type T_PBoS2ej5ec7 string
// Type-level shape guard stub; `ensure m is { field }` is not lowered to runtime checks yet.
func G_3pdR3GAa1n5(m MutationArg, ctx T_PBoS2ej5ec7) bool {
	return true
}
// Type-level shape guard stub; `ensure m is { field }` is not lowered to runtime checks yet.
func G_7gkSUCcbRH3(m MutationArg, input T_PBoS2ej5ec7) bool {
	return true
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
	println("Creating task, logged in with sessionId: " + string(*op.ctx.sessionId))
	return op.input.name, nil
}
func main() {
	sessionId := "479569ae-cbf0-471e-b849-38a698e0cb69"
	createTask(T_488eVThFocF{ctx: AppContext{sessionId: &sessionId, user: &User{}}, input: User{name: "Fix memory leak in Node.js app"}})
	println("Correctly created task (Result API)")
	createTask(T_488eVThFocF{ctx: AppContext{sessionId: nil, user: nil}, input: User{name: "Go to the gym"}})
	println("Correctly avoided creating gym task as user was not logged in")
}
