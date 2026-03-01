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
