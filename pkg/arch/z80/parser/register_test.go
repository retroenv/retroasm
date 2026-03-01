package parser

import (
	"testing"

	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestRegisterCandidatesForIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantParams []cpuz80.RegisterParam
	}{
		{
			name:       "plain register",
			value:      "A",
			wantParams: []cpuz80.RegisterParam{cpuz80.RegA},
		},
		{
			name:       "condition only",
			value:      "NZ",
			wantParams: []cpuz80.RegisterParam{cpuz80.RegCondNZ},
		},
		{
			name:       "ambiguous c yields register and condition",
			value:      "c",
			wantParams: []cpuz80.RegisterParam{cpuz80.RegC, cpuz80.RegCondC},
		},
		{
			name:       "unknown identifier",
			value:      "loop",
			wantParams: []cpuz80.RegisterParam{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := registerCandidatesForIdentifier(tt.value)
			assert.Equal(t, tt.wantParams, params)
		})
	}
}

func TestRegisterCandidatesForNumber(t *testing.T) {
	tests := []struct {
		name       string
		value      uint64
		wantParams []cpuz80.RegisterParam
	}{
		{
			name:       "im mode 0 and rst 00",
			value:      0x00,
			wantParams: []cpuz80.RegisterParam{cpuz80.RegRst00, cpuz80.RegIM0},
		},
		{
			name:       "im mode 1",
			value:      0x01,
			wantParams: []cpuz80.RegisterParam{cpuz80.RegIM1},
		},
		{
			name:       "rst 38h",
			value:      0x38,
			wantParams: []cpuz80.RegisterParam{cpuz80.RegRst38},
		},
		{
			name:       "unsupported numeric",
			value:      0x03,
			wantParams: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := registerCandidatesForNumber(tt.value)
			assert.Equal(t, tt.wantParams, params)
		})
	}
}

func TestRegisterCandidatesForIndirectIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantParams []cpuz80.RegisterParam
	}{
		{
			name:       "hl indirect",
			value:      "hl",
			wantParams: []cpuz80.RegisterParam{cpuz80.RegHLIndirect},
		},
		{
			name:       "ix parenthesized maps to ix register",
			value:      "ix",
			wantParams: []cpuz80.RegisterParam{cpuz80.RegIX},
		},
		{
			name:       "port c parenthesized",
			value:      "c",
			wantParams: []cpuz80.RegisterParam{cpuz80.RegC},
		},
		{
			name:       "unknown indirect",
			value:      "loop",
			wantParams: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := registerCandidatesForIndirectIdentifier(tt.value)
			assert.Equal(t, tt.wantParams, params)
		})
	}
}

func TestIndexedIndirectRegister(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantParam cpuz80.RegisterParam
		wantOK    bool
	}{
		{
			name:      "ix indexed",
			value:     "ix",
			wantParam: cpuz80.RegIXIndirect,
			wantOK:    true,
		},
		{
			name:      "iy indexed",
			value:     "iy",
			wantParam: cpuz80.RegIYIndirect,
			wantOK:    true,
		},
		{
			name:   "unsupported indexed register",
			value:  "hl",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param, ok := indexedIndirectRegister(tt.value)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantParam, param)
		})
	}
}
