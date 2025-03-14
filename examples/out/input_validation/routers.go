package input_validation

type UserRouter struct {
	CreateUser func(CreateUserInput) (*CreateUserOutput, error)
}

func NewUserRouter() *UserRouter {
	return &UserRouter{
		CreateUser: createUser,
	}
}
