package cpu68000

import (
	"fmt"
	"testing"

	archparser "github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

func TestAssembleFormattedDecrementBranch(t *testing.T) {
	operands := archparser.BinaryOperands(cpu68000.SizeWord,
		archparser.DataRegister(0), archparser.Absolute(true, ast.NewLabel("loop")))
	operands.Extra = 1
	instruction, err := archparser.BuildInstruction("DBF",
		cpu68000.Instructions[cpu68000.DBccName], operands)
	assert.NoError(t, err)
	formatted, err := archparser.FormatInstruction(instruction)
	assert.NoError(t, err)
	assert.Equal(t, "dbf d0,loop", formatted)
	result := assembleCPU68000(t, ".segment \"CODE\"\nloop:\nNOP\n"+formatted+"\n")
	assertBytes(t, []byte{0x4e, 0x71, 0x51, 0xc8, 0xff, 0xfc}, result)
}

func TestDecrementBranchAliasesExecute(t *testing.T) {
	for _, mnemonic := range []string{"DBF", "dbra.w", "DBT"} {
		t.Run(mnemonic, func(t *testing.T) {
			program := assembleCPU68000(t, ".segment \"CODE\"\nloop:\nNOP\n"+mnemonic+" D0,loop\n")
			memory := cpu68000.NewBasicMemory()
			for offset, value := range program {
				memory.Write(uint32(0x200+offset), value)
			}
			for _, counter := range []uint32{0, 1, 31} {
				for seed := range uint8(32) {
					t.Run(fmt.Sprintf("counter%d/flags%d", counter, seed), func(t *testing.T) {
						runDecrementBranch(t, memory, mnemonic == "DBT", counter, seed)
					})
				}
			}
		})
	}
}

func runDecrementBranch(t *testing.T, memory *cpu68000.BasicMemory, alwaysTrue bool, counter uint32, seed uint8) {
	t.Helper()
	processor, err := cpu68000.New(cpu68000.NewBasicBus(memory), cpu68000.WithInitialPC(0x200))
	assert.NoError(t, err)
	processor.D[0] = 0xa5a50000 | counter
	flags := cpu68000.Flags{
		C: seed & 1,
		V: seed >> 1 & 1,
		Z: seed >> 2 & 1,
		N: seed >> 3 & 1,
		X: seed >> 4 & 1,
	}
	processor.Flags = flags
	visits := uint32(0)
	for steps := 0; processor.PC != 0x206; steps++ {
		assert.Less(t, steps, 100)
		if processor.PC == 0x200 {
			visits++
		}
		assert.NoError(t, processor.Step())
	}
	wantCounter, wantVisits := uint32(0xa5a5ffff), counter+1
	if alwaysTrue {
		wantCounter, wantVisits = 0xa5a50000|counter, 1
	}
	assert.Equal(t, wantCounter, processor.D[0])
	assert.Equal(t, wantVisits, visits)
	assert.Equal(t, flags, processor.Flags)
}
