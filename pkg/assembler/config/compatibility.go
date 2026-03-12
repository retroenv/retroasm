// Package config provides the assembler configuration.

package config

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidCompatibilityMode indicates an unrecognized compatibility mode string.
var ErrInvalidCompatibilityMode = errors.New("invalid compatibility mode")

// CompatibilityMode selects the input syntax variant for the assembler.
type CompatibilityMode int

const (
	CompatDefault CompatibilityMode = iota // current behavior (asm6/ca65 hybrid)
	CompatX816                             // x816 assembler (65816/6502)
	CompatAsm6                             // asm6 / asm6f
	CompatCa65                             // cc65 toolchain assembler
	CompatNesasm                           // NESASM (MagicKit)
)

var compatNames = map[CompatibilityMode]string{
	CompatDefault: "default",
	CompatX816:    "x816",
	CompatAsm6:    "asm6",
	CompatCa65:    "ca65",
	CompatNesasm:  "nesasm",
}

var compatFromString = map[string]CompatibilityMode{
	"default": CompatDefault,
	"x816":    CompatX816,
	"asm6":    CompatAsm6,
	"ca65":    CompatCa65,
	"nesasm":  CompatNesasm,
}

// String returns the string representation of the compatibility mode.
func (m CompatibilityMode) String() string {
	if s, ok := compatNames[m]; ok {
		return s
	}
	return fmt.Sprintf("CompatibilityMode(%d)", int(m))
}

// ParseCompatibilityMode parses a string into a CompatibilityMode.
func ParseCompatibilityMode(s string) (CompatibilityMode, error) {
	mode, ok := compatFromString[strings.ToLower(strings.TrimSpace(s))]
	if !ok {
		return CompatDefault, fmt.Errorf("%w: '%s' (supported: default, x816, asm6, ca65, nesasm)", ErrInvalidCompatibilityMode, s)
	}
	return mode, nil
}

// ColonOptionalLabels returns whether this mode treats trailing colons on labels as optional.
func (m CompatibilityMode) ColonOptionalLabels() bool {
	return m == CompatX816 || m == CompatAsm6
}

// AnonymousLabels returns whether this mode supports +/- anonymous labels.
func (m CompatibilityMode) AnonymousLabels() bool {
	return m == CompatX816 || m == CompatAsm6
}

// AsteriskProgramCounter returns whether this mode accepts * as a program counter reference.
func (m CompatibilityMode) AsteriskProgramCounter() bool {
	return m == CompatX816 || m == CompatCa65 || m == CompatNesasm
}

// BankByteOperator returns whether this mode supports ^ as bank byte (bits 16-23) operator.
func (m CompatibilityMode) BankByteOperator() bool {
	return m == CompatX816 || m == CompatCa65
}
