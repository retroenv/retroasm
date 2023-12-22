package ast

import (
	"github.com/retroenv/assembler/expression"
)

// Base ...
type Base struct {
	node

	Address *expression.Expression
}
