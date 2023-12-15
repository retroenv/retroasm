package ast

import (
	"github.com/retroenv/assembler/expression"
)

// ConfigurationItem ...
type ConfigurationItem int

const (
	ConfigInvalid ConfigurationItem = iota
	ConfigMapper
	ConfigSubMapper
	ConfigPrg
	ConfigChr
	ConfigBattery
	ConfigMirror
	ConfigFillValue
)

// Configuration ...
type Configuration struct {
	Item       ConfigurationItem
	Value      uint64
	Expression *expression.Expression

	Comment Comment
}

func (c *Configuration) node() {}

// SetComment sets the comment for the node.
func (c *Configuration) SetComment(message string) {
	c.Comment.Message = message
}
