package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
	"github.com/retroenv/retrogolib/assert"
)

var ldhResolutionTests = []struct {
	name     string
	op1, op2 rawOperand
	register cpusm83.RegisterParam
	opcode   byte
	value    ast.Node
}{
	{
		name: "register store",
		op1: rawOperand{
			indirect: true,
			register: cpusm83.RegC,
		},
		op2:      rawOperand{register: cpusm83.RegA},
		register: cpusm83.RegCIndirect,
		opcode:   0xe2,
	},
	{
		name: "register load",
		op1:  rawOperand{register: cpusm83.RegA},
		op2: rawOperand{
			indirect: true,
			register: cpusm83.RegC,
		},
		register: cpusm83.RegLoadCIndirect,
		opcode:   0xf2,
	},
	{
		name: "immediate store",
		op1: rawOperand{
			indirect: true,
			value:    ast.NewNumber(0),
		},
		op2:      rawOperand{register: cpusm83.RegA},
		register: cpusm83.RegHighMem,
		opcode:   0xe0,
		value:    ast.NewNumber(0),
	},
	{
		name: "immediate load",
		op1:  rawOperand{register: cpusm83.RegA},
		op2: rawOperand{
			indirect: true,
			value:    ast.NewNumber(0xff),
		},
		register: cpusm83.RegLoadHighMem,
		opcode:   0xf0,
		value:    ast.NewNumber(0xff),
	},
}

func TestResolveSpecialLDH(t *testing.T) {
	for _, test := range ldhResolutionTests {
		t.Run(test.name, func(t *testing.T) {
			// The register key must select the load or store opcode.
			resolved := resolveSpecialLDH([]*cpusm83.Instruction{cpusm83.LdhInst}, test.op1, test.op2)
			assert.NotNil(t, resolved)
			assert.Equal(t, []cpusm83.RegisterParam{test.register}, resolved.RegisterParams)
			info, addressing, err := resolved.OpcodeInfo()
			assert.NoError(t, err)
			assert.Equal(t, test.opcode, info.Opcode)
			if test.value == nil {
				assert.Equal(t, cpusm83.ImpliedAddressing, addressing)
				assert.Empty(t, resolved.OperandValues)
				assert.Equal(t, 1, info.Size)
			} else {
				assert.Equal(t, cpusm83.ImmediateAddressing, addressing)
				assert.Equal(t, []ast.Node{test.value}, resolved.OperandValues)
				assert.Equal(t, 2, info.Size)
			}
			assert.Nil(t, resolveSpecialLDH([]*cpusm83.Instruction{cpusm83.LdReg8}, test.op1, test.op2))
		})
	}
}

func TestResolveSpecialLDH_InvalidRegister(t *testing.T) {
	assert.Nil(t, resolveSpecialLDH([]*cpusm83.Instruction{cpusm83.LdhInst},
		rawOperand{register: cpusm83.RegB}, rawOperand{
			indirect: true,
			register: cpusm83.RegC,
		}))
}
