package main

type T_Password = string

// Generated sub-type
type T_StrongPassword struct {
	value T_Password
}

// Generated validation function
func IsStrong(password T_Password) bool {
	return len(password) >= 12
}
