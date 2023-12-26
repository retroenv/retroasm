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
	*node

	Item       ConfigurationItem
	Value      uint64
	Expression *expression.Expression
}

// NewConfiguration returns a new configuration node.
func NewConfiguration(item ConfigurationItem) Configuration {
	return Configuration{
		node: &node{},
		Item: item,
	}
}
