package x86_test

import (
	"encoding/binary"
	"strings"
	"testing"

	asmx86 "github.com/retroenv/retroasm/pkg/arch/x86"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
	"github.com/retroenv/retrogolib/assert"
)

func TestAllConditionalBranchesRelax(t *testing.T) {
	names := []string{isa.JoName, isa.JnoName, isa.JbName, isa.JnbName, isa.JzName, isa.JnzName,
		isa.JbeName, isa.JnbeName, isa.JsName, isa.JnsName, isa.JpName, isa.JnpName,
		isa.JlName, isa.JnlName, isa.JleName, isa.JnleName}
	for index, name := range names {
		for _, distance := range []int{-129, -128, 127, 128} {
			c := newCodec(t)
			instruction, err := asmx86.NewInstruction(name, ast.NewNumber(uint64(0x102+distance)))
			assert.NoError(t, err)
			assembly, err := c.Assemble(t.Context(), []ast.Node{ast.NewSegment("CODE"), instruction})
			assert.NoError(t, err)
			opcode := byte(0x70 + index)
			if distance >= -128 && distance <= 127 {
				assert.Equal(t, []byte{opcode, byte(distance)}, assembly.Binary)
			} else {
				expected := binary.LittleEndian.AppendUint16([]byte{opcode ^ 1, 3, 0xe9}, uint16(distance-3))
				assert.Equal(t, expected, assembly.Binary)
			}
		}
	}
}

func TestConditionalBranchRelaxationRepeatsLayout(t *testing.T) {
	c := newCodec(t)
	nodes, err := c.Parse(t.Context(), strings.NewReader(`.segment "CODE"
jz near_target
jz far_target
.dsb 125, 0
near_target:
.dsb 200, 0
far_target:
ret
`))
	assert.NoError(t, err)
	first, err := c.Assemble(t.Context(), nodes)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0x187), first.Symbols["near_target"])
	assert.Equal(t, uint64(0x24f), first.Symbols["far_target"])
	assert.Equal(t, []byte{0x75, 3, 0xe9, 0x82, 0, 0x75, 3, 0xe9, 0x45, 1}, first.Binary[:10])
	second, err := c.Assemble(t.Context(), nodes)
	assert.NoError(t, err)
	assert.Equal(t, first.Binary, second.Binary)
	assert.Equal(t, first.Symbols, second.Symbols)
}
