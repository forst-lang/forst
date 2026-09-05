package store

type Store struct {
	N int
}

func (s *Store) Get() int {
	if s == nil {
		return 0
	}
	return s.N
}
