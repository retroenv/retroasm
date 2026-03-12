package chip8

import (
	"bytes"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retrogolib/assert"
)

// --- Implied addressing (no operands) ---

func TestAssembleChip8_Cls(t *testing.T) {
	out := assembleChip8Source(t, "cls\n")
	assert.Equal(t, []byte{0x00, 0xE0}, out)
}

func TestAssembleChip8_Ret(t *testing.T) {
	out := assembleChip8Source(t, "ret\n")
	assert.Equal(t, []byte{0x00, 0xEE}, out)
}

// --- Absolute addressing ---

func TestAssembleChip8_JpAbsolute(t *testing.T) {
	out := assembleChip8Source(t, "jp $300\n")
	assert.Equal(t, []byte{0x13, 0x00}, out)
}

func TestAssembleChip8_JpLabel(t *testing.T) {
	src := `
jp target
cls
target:
cls
`
	out := assembleChip8Source(t, src)
	// jp target ($204) → 0x1204, cls → 0x00E0, cls → 0x00E0
	assert.Equal(t, []byte{0x12, 0x04, 0x00, 0xE0, 0x00, 0xE0}, out)
}

func TestAssembleChip8_CallAbsolute(t *testing.T) {
	out := assembleChip8Source(t, "call $400\n")
	assert.Equal(t, []byte{0x24, 0x00}, out)
}

func TestAssembleChip8_CallLabel(t *testing.T) {
	src := `
call sub
cls
sub:
ret
`
	out := assembleChip8Source(t, src)
	// call sub ($204) → 0x2204, cls → 0x00E0, ret → 0x00EE
	assert.Equal(t, []byte{0x22, 0x04, 0x00, 0xE0, 0x00, 0xEE}, out)
}

// --- V0 + absolute addressing ---

func TestAssembleChip8_JpV0Absolute(t *testing.T) {
	out := assembleChip8Source(t, "jp v0, $300\n")
	assert.Equal(t, []byte{0xB3, 0x00}, out)
}

func TestAssembleChip8_JpV0Label(t *testing.T) {
	src := `
jp v0, target
target:
cls
`
	out := assembleChip8Source(t, src)
	// jp v0, target ($202) → 0xB202, cls → 0x00E0
	assert.Equal(t, []byte{0xB2, 0x02, 0x00, 0xE0}, out)
}

// --- SE (skip if equal) ---

func TestAssembleChip8_SeRegisterValue(t *testing.T) {
	out := assembleChip8Source(t, "se v1, $42\n")
	assert.Equal(t, []byte{0x31, 0x42}, out)
}

func TestAssembleChip8_SeRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "se v1, v2\n")
	assert.Equal(t, []byte{0x51, 0x20}, out)
}

// --- SNE (skip if not equal) ---

func TestAssembleChip8_SneRegisterValue(t *testing.T) {
	out := assembleChip8Source(t, "sne v3, $10\n")
	assert.Equal(t, []byte{0x43, 0x10}, out)
}

func TestAssembleChip8_SneRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "sne va, vb\n")
	assert.Equal(t, []byte{0x9A, 0xB0}, out)
}

// --- LD (load) - all addressing modes ---

func TestAssembleChip8_LdRegisterValue(t *testing.T) {
	out := assembleChip8Source(t, "ld v3, $7f\n")
	assert.Equal(t, []byte{0x63, 0x7F}, out)
}

func TestAssembleChip8_LdRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "ld v4, v5\n")
	assert.Equal(t, []byte{0x84, 0x50}, out)
}

func TestAssembleChip8_LdIAbsolute(t *testing.T) {
	out := assembleChip8Source(t, "ld i, $300\n")
	assert.Equal(t, []byte{0xA3, 0x00}, out)
}

func TestAssembleChip8_LdILabel(t *testing.T) {
	src := `
ld i, sprite
cls
sprite:
cls
`
	out := assembleChip8Source(t, src)
	// ld i, sprite ($204) → 0xA204, cls → 0x00E0, cls → 0x00E0
	assert.Equal(t, []byte{0xA2, 0x04, 0x00, 0xE0, 0x00, 0xE0}, out)
}

func TestAssembleChip8_LdRegisterDT(t *testing.T) {
	out := assembleChip8Source(t, "ld v2, dt\n")
	assert.Equal(t, []byte{0xF2, 0x07}, out)
}

func TestAssembleChip8_LdRegisterK(t *testing.T) {
	out := assembleChip8Source(t, "ld v0, k\n")
	assert.Equal(t, []byte{0xF0, 0x0A}, out)
}

func TestAssembleChip8_LdDTRegister(t *testing.T) {
	out := assembleChip8Source(t, "ld dt, v5\n")
	assert.Equal(t, []byte{0xF5, 0x15}, out)
}

func TestAssembleChip8_LdSTRegister(t *testing.T) {
	out := assembleChip8Source(t, "ld st, v3\n")
	assert.Equal(t, []byte{0xF3, 0x18}, out)
}

func TestAssembleChip8_LdFRegister(t *testing.T) {
	out := assembleChip8Source(t, "ld f, v7\n")
	assert.Equal(t, []byte{0xF7, 0x29}, out)
}

func TestAssembleChip8_LdBRegister(t *testing.T) {
	out := assembleChip8Source(t, "ld b, va\n")
	assert.Equal(t, []byte{0xFA, 0x33}, out)
}

func TestAssembleChip8_LdIndirectIRegister(t *testing.T) {
	out := assembleChip8Source(t, "ld [i], v1\n")
	assert.Equal(t, []byte{0xF1, 0x55}, out)
}

func TestAssembleChip8_LdRegisterIndirectI(t *testing.T) {
	out := assembleChip8Source(t, "ld v1, [i]\n")
	assert.Equal(t, []byte{0xF1, 0x65}, out)
}

// --- ADD - all addressing modes ---

func TestAssembleChip8_AddRegisterValue(t *testing.T) {
	out := assembleChip8Source(t, "add v1, $10\n")
	assert.Equal(t, []byte{0x71, 0x10}, out)
}

func TestAssembleChip8_AddRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "add v1, v2\n")
	assert.Equal(t, []byte{0x81, 0x24}, out)
}

func TestAssembleChip8_AddIRegister(t *testing.T) {
	out := assembleChip8Source(t, "add i, v1\n")
	assert.Equal(t, []byte{0xF1, 0x1E}, out)
}

// --- Bitwise operations (register-register) ---

func TestAssembleChip8_OrRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "or v1, v2\n")
	assert.Equal(t, []byte{0x81, 0x21}, out)
}

func TestAssembleChip8_AndRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "and v3, v4\n")
	assert.Equal(t, []byte{0x83, 0x42}, out)
}

func TestAssembleChip8_XorRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "xor v5, v6\n")
	assert.Equal(t, []byte{0x85, 0x63}, out)
}

// --- Arithmetic (register-register) ---

func TestAssembleChip8_SubRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "sub v1, v2\n")
	assert.Equal(t, []byte{0x81, 0x25}, out)
}

func TestAssembleChip8_SubnRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "subn v3, v4\n")
	assert.Equal(t, []byte{0x83, 0x47}, out)
}

// --- Shift operations ---

func TestAssembleChip8_ShrRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "shr v1, v2\n")
	assert.Equal(t, []byte{0x81, 0x26}, out)
}

func TestAssembleChip8_ShlRegisterRegister(t *testing.T) {
	out := assembleChip8Source(t, "shl v1, v2\n")
	assert.Equal(t, []byte{0x81, 0x2E}, out)
}

// --- RND ---

func TestAssembleChip8_RndRegisterValue(t *testing.T) {
	out := assembleChip8Source(t, "rnd v1, $ff\n")
	assert.Equal(t, []byte{0xC1, 0xFF}, out)
}

// --- DRW ---

func TestAssembleChip8_DrwRegisterRegisterNibble(t *testing.T) {
	out := assembleChip8Source(t, "drw v1, v2, 5\n")
	assert.Equal(t, []byte{0xD1, 0x25}, out)
}

func TestAssembleChip8_DrwNibbleIdentifier(t *testing.T) {
	src := `
nib = 3
drw v1, v2, nib
`
	out := assembleChip8Source(t, src)
	assert.Equal(t, []byte{0xD1, 0x23}, out)
}

// --- SKP / SKNP ---

func TestAssembleChip8_SkpRegister(t *testing.T) {
	out := assembleChip8Source(t, "skp v1\n")
	assert.Equal(t, []byte{0xE1, 0x9E}, out)
}

func TestAssembleChip8_SknpRegister(t *testing.T) {
	out := assembleChip8Source(t, "sknp vf\n")
	assert.Equal(t, []byte{0xEF, 0xA1}, out)
}

// --- High register values (V8-VF) ---

func TestAssembleChip8_HighRegisters(t *testing.T) {
	src := `
ld va, $11
ld vf, $22
or vc, vd
add ve, v8
`
	out := assembleChip8Source(t, src)
	assert.Equal(t, []byte{
		0x6A, 0x11, // ld va, $11
		0x6F, 0x22, // ld vf, $22
		0x8C, 0xD1, // or vc, vd
		0x8E, 0x84, // add ve, v8
	}, out)
}

// --- Identifier/symbol references ---

func TestAssembleChip8_RegisterValueIdentifier(t *testing.T) {
	src := `
val = $7f
ld v3, val
`
	out := assembleChip8Source(t, src)
	assert.Equal(t, []byte{0x63, 0x7F}, out)
}

func TestAssembleChip8_RegisterValueIdentifierAndV0Jump(t *testing.T) {
	src := `
val = $7f
jp v0, target
ld v3, val
target:
cls
`
	out := assembleChip8Source(t, src)
	assert.Equal(t, []byte{0xB2, 0x04, 0x63, 0x7F, 0x00, 0xE0}, out)
}

// --- Combined program test ---

func TestAssembleChip8_AllInstructionsCombined(t *testing.T) {
	src := `
cls
ld i, $300
ld v0, $0a
ld v1, $14
add v0, $01
se v0, v1
jp loop
ld dt, v0
ld st, v1
ld v2, dt
ld v3, k
ld f, v0
ld b, v1
ld [i], v2
ld v3, [i]
add i, v0
drw v0, v1, 5
skp v0
sknp v1
rnd v0, $ff
or v0, v1
and v0, v1
xor v0, v1
sub v0, v1
subn v0, v1
shr v0, v1
shl v0, v1
sne v0, $10
sne v0, v1
se v0, $10
call sub1
loop:
jp loop
sub1:
ret
`
	out := assembleChip8Source(t, src)
	assert.Equal(t, []byte{
		0x00, 0xE0, // cls
		0xA3, 0x00, // ld i, $300
		0x60, 0x0A, // ld v0, $0a
		0x61, 0x14, // ld v1, $14
		0x70, 0x01, // add v0, $01
		0x50, 0x10, // se v0, v1
		0x12, 0x3E, // jp loop ($23E)
		0xF0, 0x15, // ld dt, v0
		0xF1, 0x18, // ld st, v1
		0xF2, 0x07, // ld v2, dt
		0xF3, 0x0A, // ld v3, k
		0xF0, 0x29, // ld f, v0
		0xF1, 0x33, // ld b, v1
		0xF2, 0x55, // ld [i], v2
		0xF3, 0x65, // ld v3, [i]
		0xF0, 0x1E, // add i, v0
		0xD0, 0x15, // drw v0, v1, 5
		0xE0, 0x9E, // skp v0
		0xE1, 0xA1, // sknp v1
		0xC0, 0xFF, // rnd v0, $ff
		0x80, 0x11, // or v0, v1
		0x80, 0x12, // and v0, v1
		0x80, 0x13, // xor v0, v1
		0x80, 0x15, // sub v0, v1
		0x80, 0x17, // subn v0, v1
		0x80, 0x16, // shr v0, v1
		0x80, 0x1E, // shl v0, v1
		0x40, 0x10, // sne v0, $10
		0x90, 0x10, // sne v0, v1
		0x30, 0x10, // se v0, $10
		0x22, 0x40, // call sub1 ($240)
		0x12, 0x3E, // jp loop ($23E)
		0x00, 0xEE, // ret
	}, out)
}

func assembleChip8Source(t *testing.T, source string) []byte {
	t.Helper()

	cfg := New()
	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	err := asm.Process(t.Context(), strings.NewReader(source))
	assert.NoError(t, err)

	return buf.Bytes()
}
