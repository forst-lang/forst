package main

type Builder struct{}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) Pair() (string, []any) {
	return "a", []any{"b"}
}
