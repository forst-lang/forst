package main

type UserRouter struct {
	CreateUser func(CreateUserInput) (*CreateUserOutput, error)
}

func NewUserRouter() *UserRouter {
	return &UserRouter{
		CreateUser: createUser,
	}
}
