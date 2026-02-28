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
