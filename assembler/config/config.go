// Package config provides the assembler configuration.
package config

import (
	"github.com/retroenv/assembler/arch"
	"github.com/retroenv/assembler/expression"
)

// Config defines an assembler config.
type Config struct {
	Arch            arch.Architecture
	Segments        map[string]*Segment
	SegmentsOrdered []*Segment

	// assembly code configurable items
	FillValues *expression.Expression
}

// Reset resets the configuration items that can be configured in the
// assembly code.
func (c *Config) Reset() {
	c.FillValues = nil
}

// Memory contains the basic configuration for a memory segment.
type Memory struct {
	Name string

	Start uint64
	Size  uint64

	Typ  string // TODO: support
	File string

	Fill      bool
	FillValue byte
}

// Segment contains the extended configuration for a memory segment.
type Segment struct {
	Memory

	SegmentName  string
	SegmentStart uint64

	Offset uint64 // TODO: support
	Align  uint64 // TODO: support
	Run    string // TODO: support

	Define   bool // TODO: support
	Optional bool // TODO: support
}
