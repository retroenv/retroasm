package config

import (
	"bytes"
	"testing"

	"github.com/retroenv/retrogolib/arch/cpu/m6502"
	"github.com/retroenv/retrogolib/assert"
)

var ca65Config = []byte(`
  # Start of memory section
MEMORY
{
    BANK_0: start = $8000, size = $4000, type = ro, fill = yes, fillval = $FF;
	RAM1: 
		start $0800
		size $9800;
}

SEGMENTS {
	CODE:   load = BANK_0, type = ro;
	DATA:   load = RAM1, type = rw;
	BSS:    load = RAM1, type = bss, define = yes;
}
`)

func TestConfigReadCa65Config(t *testing.T) {
	reader := bytes.NewReader(ca65Config)
	var cfg Config[*m6502.Instruction]
	assert.NoError(t, cfg.ReadCa65Config(reader))
}
