package ensure

type TooShort struct {
	message string
}

func (e TooShort) Error() string {
	return e.message
}

func assertion(name string) error {
	if len(name) == 0 {
		return TooShort{"Name must be at least 1 character long"}
	}
	return nil
}
