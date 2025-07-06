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
// T_5fHJb69t288: TypeDefShapeExpr({ctx: AppContext., input: T_GVKpDX7vGyx})
type T_5fHJb69t288 struct {
        ctx   AppContext
        input T_GVKpDX7vGyx
}
// T_7zywpPhwVhj: TypeDefShapeExpr({name: Value("Alice")})
type T_7zywpPhwVhj struct {
        name string
}
// T_DBy8B2VnfFA: TypeDefShapeExpr({input: Shape})
type T_DBy8B2VnfFA struct {
        input struct{}
}
// T_DX1exfNyf6L: TypeDefShapeExpr({name: Value("Fix memory leak in Node.js app")})
type T_DX1exfNyf6L struct {
        name string
}
// T_GVKpDX7vGyx: TypeDefShapeExpr({name: String})
type T_GVKpDX7vGyx struct {
        name string
}
// T_HFDHMNeRC3v: TypeDefShapeExpr({input: Shape})
type T_HFDHMNeRC3v struct {
        input struct{}
}
// T_LYQafBLM8TQ: TypeDefShapeExpr({sessionId: Value(Ref(Variable(sessionId))), user: {name: Value("Alice")}})
type T_LYQafBLM8TQ struct {
        sessionId *string
        user      T_7zywpPhwVhj
}
// T_X86jJwVQ4mH: TypeDefShapeExpr({sessionId: Value(nil)})
type T_X86jJwVQ4mH struct {
        sessionId string
}
// T_Yo2G7tbwv5q: TypeDefShapeExpr({ctx: {sessionId: Value(Ref(Variable(sessionId))), user: {name: Value("Alice")}}, input: {name: Value("Fix memory leak in Node.js app")}})
type T_Yo2G7tbwv5q struct {
        ctx   T_LYQafBLM8TQ
        input T_DX1exfNyf6L
}
// T_azh9nsqmxaF: TypeDefShapeExpr({name: String})
type T_azh9nsqmxaF struct {
        name string
}
// T_b7WdLNpLZoy: TypeDefShapeExpr({ctx: {sessionId: Value(nil)}, input: {name: Value("Go to the gym")}})
type T_b7WdLNpLZoy struct {
        ctx   T_X86jJwVQ4mH
        input T_goedQ5siUEo
}
// T_bN37LzcUYJ8: TypeDefAssertionExpr(Value(nil))
type T_bN37LzcUYJ8 string
// T_goedQ5siUEo: TypeDefShapeExpr({name: Value("Go to the gym")})
type T_goedQ5siUEo struct {
        name string
}
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
func createTask(op T_5fHJb69t288) (string, error) {
        if !G_W5HnAr5D9X1(op.ctx) {
                return "", errors.New("assertion failed: " + "AppContext.LoggedIn()")
        }
        println("Creating task, logged in with sessionId: " + *op.ctx.sessionId)
        return op.input.name, nil
}
func main() {
        sessionId := "479569ae-cbf0-471e-b849-38a698e0cb69"
        name, err := createTask(T_5fHJb69t288{ctx: AppContext{sessionId: &sessionId, user: &User{name: "Alice"}}, input: T_GVKpDX7vGyx{name: "Fix memory leak in Node.js app"}})
        if err != nil {
                println(err.Error())
                panic(errors.New("assertion failed: " + "Error.Nil()"))
        }
        println("Correctly created task: " + name)
        name, err = createTask(T_5fHJb69t288{ctx: AppContext{sessionId: nil}, input: T_GVKpDX7vGyx{name: "Go to the gym"}})
        if err == nil {
                println("Expected error but gym task was created")
                panic(errors.New("assertion failed: " + "Error.Present()"))
        }
        println("Correctly avoided creating gym task as user was not logged in")
}