package assembler

import (
	"bytes"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retrogolib/assert"
)

func TestAssemblerX816ForwardLabelAlias(t *testing.T) {
	const code = `
.segment "HEADER"

TargetOffset = target - 1
lda TargetOffset
target:
nop
`

	cfg := m6502.New()
	cfg.CompatibilityMode = config.CompatX816
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	var output bytes.Buffer
	asm := New(cfg, &output)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(code)))
	assert.Equal(t, []byte{0xad, 0x02, 0x00, 0xea}, output.Bytes())
}

func TestAssemblerX816ForwardAddressByteData(t *testing.T) {
	const code = `
.segment "HEADER"

.db <first, <second
.db >first, >second
first:
nop
second:
nop
`

	cfg := m6502.New()
	cfg.CompatibilityMode = config.CompatX816
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	var output bytes.Buffer
	asm := New(cfg, &output)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(code)))
	assert.Equal(t, []byte{0x04, 0x05, 0x00, 0x00, 0xea, 0xea}, output.Bytes())
}

func TestAssemblerX816ForwardAddressData(t *testing.T) {
	const code = `
.segment "HEADER"

.dw first, second
first:
nop
second:
nop
`

	cfg := m6502.New()
	cfg.CompatibilityMode = config.CompatX816
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	var output bytes.Buffer
	asm := New(cfg, &output)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(code)))
	assert.Equal(t, []byte{0x04, 0x00, 0x05, 0x00, 0xea, 0xea}, output.Bytes())
}

func TestAssemblerX816ForwardDataExpression(t *testing.T) {
	const code = `
.segment "HEADER"

.db second-first, second-first
first:
nop
second:
nop
`

	cfg := m6502.New()
	cfg.CompatibilityMode = config.CompatX816
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	var output bytes.Buffer
	asm := New(cfg, &output)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(code)))
	assert.Equal(t, []byte{0x01, 0x01, 0xea, 0xea}, output.Bytes())
}

func TestAssemblerX816NumericIndirectJump(t *testing.T) {
	const code = `
.segment "HEADER"
before:
jmp ($06)
after:
nop
`

	cfg := m6502.New()
	cfg.CompatibilityMode = config.CompatX816
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	var output bytes.Buffer
	asm := New(cfg, &output)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(code)))
	// Bug: missing indirect-JMP size metadata made the following instruction overwrite its opcode.
	assert.Equal(t, []byte{0x6c, 0x06, 0x00, 0xea}, output.Bytes())
	assert.Equal(t, uint64(0), asm.Symbols()["before"])
	assert.Equal(t, uint64(3), asm.Symbols()["after"])
}

func TestAssemblerX816AliasDataList(t *testing.T) {
	const code = `
.segment "HEADER"
Water = 2
Ground = 1
Underground = 4
Castle = 8
Cloud = 16
Pipe = 32
.db Water, Ground, Underground, Castle
.db Cloud, Pipe
`

	cfg := m6502.New()
	cfg.CompatibilityMode = config.CompatX816
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	var output bytes.Buffer
	asm := New(cfg, &output)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(code)))
	// Bug: resolved aliases in lists were encoded as their decimal text instead of byte values.
	assert.Equal(t, []byte{2, 1, 4, 8, 16, 32}, output.Bytes())
}
