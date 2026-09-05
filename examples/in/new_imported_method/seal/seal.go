package seal

type Seal struct {
	Mark string
}

func NewSeal(mark string) *Seal {
	return &Seal{Mark: mark}
}
