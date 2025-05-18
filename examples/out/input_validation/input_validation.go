package input_validation

import fmt "fmt"

// TypeDefAssertionExpr(Shape({iban: TYPE_STRING.Min(10).Max(34)}))
type T_F22VkNXdy8p struct {
	iban T_hK5bA1s8g6u
}

// TypeDefAssertionExpr(Shape({name: TYPE_STRING.Min(3).Max(10), phoneNumber: PhoneNumber, bankAccount: {iban: TYPE_STRING.Min(10).Max(34)}, id: UUID.V4()}))
type T_3MopQmWpjaX struct {
	phoneNumber T_TVaWFGLH1EY
	bankAccount T_F22VkNXdy8p
	id          T_eYQHwKxPtJU
	name        T_duDtN9av3sv
}

// TypeDefAssertionExpr(PhoneNumber)
type T_YMCMSbyueZo PhoneNumber

// TypeDefAssertionExpr(TYPE_STRING.Min(10).Max(34))
type T_hK5bA1s8g6u string

// TypeDefAssertionExpr(TYPE_STRING.Min(10).Max(34))
type T_Sc5Sb6v9EnC string

// TypeDefBinaryExpr(TypeDefAssertionExpr(TYPE_STRING.Min(3).Max(10)) & TypeDefBinaryExpr(TypeDefAssertionExpr(TYPE_STRING.HasPrefix("+")) | TypeDefAssertionExpr(TYPE_STRING.HasPrefix("0"))))
type PhoneNumber string

// TypeDefAssertionExpr(trpc.Mutation.Input({bankAccount: {iban: TYPE_STRING.Min(10).Max(34)}, id: UUID.V4(), name: TYPE_STRING.Min(3).Max(10), phoneNumber: PhoneNumber}))
type T_KS693MHKmJw struct {
	ctx struct {
	}
	input struct {
		id          T_eYQHwKxPtJU
		name        T_duDtN9av3sv
		phoneNumber T_TVaWFGLH1EY
		bankAccount T_F22VkNXdy8p
	}
}

// TypeDefAssertionExpr(PhoneNumber)
type T_TVaWFGLH1EY PhoneNumber

// TypeDefAssertionExpr(UUID.V4())
type T_eYQHwKxPtJU string

// TypeDefAssertionExpr(UUID.V4())
type T_2jb6kBbD7Nn string

// TypeDefAssertionExpr(TYPE_STRING.Min(3).Max(10))
type T_duDtN9av3sv string

// TypeDefAssertionExpr(TYPE_STRING.Min(3).Max(10))
type T_WtzqCu9trFG string

func createUser(op T_T41S7Ud4yaM) float64 {
	fmt.Println("Creating user with id: %s", op.input.id)
	return 300.3
}
