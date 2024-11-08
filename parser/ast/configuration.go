package ast

import (
	"github.com/retroenv/retroasm/expression"
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

// Copy returns a copy of the configuration node.
func (c Configuration) Copy() Node {
	return Configuration{
		node:       c.node,
		Item:       c.Item,
		Value:      c.Value,
		Expression: c.Expression.Copy(),
	}
}
