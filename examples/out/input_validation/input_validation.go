package input_validation

import fmt "fmt"
// TypeDefBinaryExpr(TypeDefAssertionExpr(TYPE_STRING.Min(3).Max(10)) & TypeDefBinaryExpr(TypeDefAssertionExpr(TYPE_STRING.HasPrefix("+")) | TypeDefAssertionExpr(TYPE_STRING.HasPrefix("0"))))
type PhoneNumber string
// TypeDefAssertionExpr(Shape({iban: TYPE_STRING.Min(10).Max(34)}))
type T_2o2q3CRbwEC struct {
	iban T_MZjxvVSFAUF
}
// TypeDefAssertionExpr(Shape({name: TYPE_STRING.Min(3).Max(10), phoneNumber: PhoneNumber, bankAccount: {iban: TYPE_STRING.Min(10).Max(34)}, id: UUID.V4()}))
type T_3z5Q2xHXNiR struct {
	bankAccount T_2o2q3CRbwEC
	id          T_e63jpC4qrag
	name        T_FBmkfDvDYKk
	phoneNumber T_BFj1JD17i2B
}
// TypeDefAssertionExpr(PhoneNumber)
type T_BFj1JD17i2B PhoneNumber
// TypeDefAssertionExpr(TYPE_STRING.Min(3).Max(10))
type T_FBmkfDvDYKk string
// TypeDefAssertionExpr(trpc.Mutation.Input({id: UUID.V4(), name: TYPE_STRING.Min(3).Max(10), phoneNumber: PhoneNumber, bankAccount: {iban: TYPE_STRING.Min(10).Max(34)}}))
type T_M8WRQEPuLtT struct {
	ctx struct {
	}
	input struct {
		bankAccount T_2o2q3CRbwEC
		id          T_e63jpC4qrag
		name        T_FBmkfDvDYKk
		phoneNumber T_BFj1JD17i2B
	}
}
// TypeDefAssertionExpr(PhoneNumber)
type T_MTvkfV8KyPY PhoneNumber
// TypeDefAssertionExpr(TYPE_STRING.Min(10).Max(34))
type T_MZjxvVSFAUF string
// TypeDefAssertionExpr(UUID.V4())
type T_ReagKMhcoCm string
// TypeDefAssertionExpr(TYPE_STRING.Min(3).Max(10))
type T_TfaC5mZgieD string
// TypeDefAssertionExpr(TYPE_STRING.Min(10).Max(34))
type T_dsCvswfU8Jb string
// TypeDefAssertionExpr(UUID.V4())
type T_e63jpC4qrag string

func createUser(op T_M8WRQEPuLtT) float64 {
	fmt.Println("Creating user with id: %s", op.input.id)
	return 300.3
}
