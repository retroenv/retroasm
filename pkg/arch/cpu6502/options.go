package cpu6502

import "github.com/retroenv/retrogolib/arch/cpu/cpu6502"

// Option configures the CPU6502 assembler architecture.
type Option func(*options)

type options struct {
	variant       cpu6502.CPUVariant
	strictVariant bool
}

// WithVariant restricts parsing, building, validation, and encoding to one
// retrogolib CPU variant. New without this option retains the historical
// compatibility registry.
func WithVariant(variant cpu6502.CPUVariant) Option {
	return func(options *options) {
		options.variant = variant
		options.strictVariant = true
	}
}
