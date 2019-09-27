package cmp

import (
	"fmt"
)

type PathWithLevel struct {
	Path   string
	Level  int
	Line   int
	Stream bool
}

func (p PathWithLevel) ID() string {
	return fmt.Sprintf("%t%s", p.Stream, p.Path)
}
