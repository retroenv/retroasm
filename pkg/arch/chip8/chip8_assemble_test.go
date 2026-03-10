package chip8

import (
	"bytes"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retrogolib/assert"
)

func assembleChip8Source(t *testing.T, source string) []byte {
	t.Helper()

	cfg := New()
	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	err := asm.Process(t.Context(), strings.NewReader(source))
	assert.NoError(t, err)

	return buf.Bytes()
}

func TestAssembleChip8_IndirectIAddressing(t *testing.T) {
	src := `
ld [i], v1
ld v1, [i]
`

	out := assembleChip8Source(t, src)
	assert.Equal(t, []byte{0xF1, 0x55, 0xF1, 0x65}, out)
}

func TestAssembleChip8_DrwNibbleIdentifier(t *testing.T) {
	src := `
nib = 3
drw v1, v2, nib
`

	out := assembleChip8Source(t, src)
	assert.Equal(t, []byte{0xD1, 0x23}, out)
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

func TestAssembleChip8_KeypadSingleRegisterInstructions(t *testing.T) {
	src := `
skp v1
sknp vf
`

	out := assembleChip8Source(t, src)
	assert.Equal(t, []byte{0xE1, 0x9E, 0xEF, 0xA1}, out)
}
