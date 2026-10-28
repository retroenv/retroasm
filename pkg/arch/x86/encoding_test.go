package x86_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	asmx86 "github.com/retroenv/retroasm/pkg/arch/x86"
	"github.com/retroenv/retroasm/pkg/arch/x86/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
	"github.com/retroenv/retrogolib/assert"
)

func TestMemoryEncodingsMatchNASM(t *testing.T) {
	for _, address := range []string{"bx+si", "bx+di", "bp+si", "bp+di", "si", "di", "bp", "bx", ""} {
		t.Run(address, func(t *testing.T) {
			var source strings.Builder
			for _, displacement := range []int{0, 1, -1, -128, 127, -129, 128, -32768, 32767, 65535} {
				if address == "" && displacement < 0 {
					continue
				}
				value := address
				if value == "" {
					value = strconv.Itoa(displacement)
				} else if displacement != 0 {
					value += fmt.Sprintf("%+d", displacement)
				}
				for _, segment := range []string{"", "es:", "cs:", "ss:", "ds:"} {
					for register := isa.RegAL; register <= isa.RegDI; register++ {
						memory := "[" + segment + value + "]"
						fmt.Fprintf(&source, "mov %s, %s\nmov %s, %s\n", register, memory, memory, register)
					}
				}
			}
			checkNASM(t, source.String(), source.String())
		})
	}
}

func TestOperationEncodingsMatchNASM(t *testing.T) {
	for _, name := range []string{isa.AddName, isa.AdcName, isa.SubName, isa.SbbName, isa.AndName,
		isa.OrName, isa.XorName, isa.CmpName, isa.TestName, isa.XchgName} {
		t.Run(name, func(t *testing.T) {
			var source strings.Builder
			for destination := isa.RegAL; destination <= isa.RegDI; destination++ {
				for origin := isa.RegAL; origin <= isa.RegDI; origin++ {
					if origin.GetRegisterSize() == destination.GetRegisterSize() {
						fmt.Fprintf(&source, "%s %s, %s\n", name, destination, origin)
					}
				}
				fmt.Fprintf(&source, "%s %s, [bp-2]\n%s [es:bx+di+128], %s\n", name, destination, name, destination)
				if name != isa.XchgName {
					value := "0xab"
					if destination.Is16Bit() {
						value = "0xabcd"
					}
					fmt.Fprintf(&source, "%s %s, %s\n", name, destination, value)
				}
			}
			if name != isa.XchgName {
				fmt.Fprintf(&source, "%s byte [bx], -1\n%s word [bp-128], 0xabcd\n", name, name)
			}
			checkNASM(t, source.String(), source.String())
		})
	}
}

func TestUnaryAndShiftEncodingsMatchNASM(t *testing.T) {
	for _, name := range []string{isa.IncName, isa.DecName, isa.NegName, isa.NotName,
		isa.MulName, isa.ImulName, isa.DivName, isa.IdivName, isa.ShlName, isa.ShrName, isa.SarName,
		isa.RolName, isa.RorName, isa.RclName, isa.RcrName} {
		t.Run(name, func(t *testing.T) {
			var source strings.Builder
			counts := []string{""}
			if name == isa.ShlName || name == isa.ShrName || name == isa.SarName || name == isa.RolName ||
				name == isa.RorName || name == isa.RclName || name == isa.RcrName {

				counts = []string{", 1", ", cl"}
			}
			for _, count := range counts {
				for register := isa.RegAL; register <= isa.RegDI; register++ {
					fmt.Fprintf(&source, "%s %s%s\n", name, register, count)
				}
				for _, memory := range []string{"byte [bx]", "word [bp]", "byte [es:bp+si-129]", "word [65535]"} {
					fmt.Fprintf(&source, "%s %s%s\n", name, memory, count)
				}
			}
			checkNASM(t, source.String(), source.String())
		})
	}
}

func TestStackStringsAndMemorySymbolsMatchNASM(t *testing.T) {
	source := `push es
push cs
push ss
push ds
pop ds
pop ss
pop es
push word [bp-2]
pop word [es:bx+di+128]
call ax
jmp word [bp-2]
mov ds, [pointer]
mov [bx+si], es
mov byte [pointer], -128
mov word [pointer+2], -32768
mov ax, -1
lea si, [bx+di-2]
clc
stc
cmc
std
cld
cbw
cwd
ret 4
iret
movsb
movsw
cmpsb
cmpsw
stosb
stosw
lodsb
lodsw
scasb
scasw
rep movsb
rep movsw
rep stosb
rep lodsw
repz cmpsb
repnz scasw
pointer:
.word 0x1234, 0x5678
`
	checkNASM(t, source, strings.ReplaceAll(source, ".word", "dw"))
}

func TestMemoryOperandRoundTripAndSymbols(t *testing.T) {
	c := newCodec(t)
	for _, text := range []string{"mov ax, word [es:bp-2]", "mov byte [buffer+2], -1", "lea di, [bx+si+128]", "rep movsw"} {
		first, err := c.ParseInstruction(t.Context(), strings.NewReader(text))
		assert.NoError(t, err)
		formatted, err := c.FormatInstruction(first)
		assert.NoError(t, err)
		assert.Equal(t, text, formatted)
		second, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
		assert.NoError(t, err)
		assert.Equal(t, first.OpcodeID, second.OpcodeID)
		assert.Equal(t, ast.InstructionArgumentForm(first.Argument), ast.InstructionArgumentForm(second.Argument))
	}
	memory := parser.MemoryOperand{
		Address:      isa.RegBPRef,
		Width:        2,
		Segment:      isa.RegES,
		Symbol:       "before",
		Displacement: -2,
	}
	instruction, err := asmx86.NewInstruction(isa.MovName, ast.NewInstructionArgument(isa.RegAX), ast.NewInstructionArgument(memory))
	assert.NoError(t, err)
	renamed, err := ast.RewriteNodeSymbols(instruction, renameX86Source)
	assert.NoError(t, err)
	text, err := c.FormatInstruction(renamed.(ast.Instruction))
	assert.NoError(t, err)
	assert.Equal(t, "mov ax, word [es:bp+renamed_before-2]", text)
	assert.Equal(t, "before", memory.Symbol)
	assert.Len(t, memory.InstructionReferences(), 1)
}

func TestRejectsInvalidMemoryAndNewForms(t *testing.T) {
	for _, text := range []string{"mov ax, [bx+bp]", "mov ax, [si+di]", "mov ax, [al]", "mov ax, [sp]",
		"mov ax, [bx+bx]", "mov ax, []", "mov ax, [bx+]", "mov ax, [bx-ax]", "mov ax, [bx*2]",
		"mov ax, [foo+bar]", "mov ax, [bx+65536]", "mov ax, byte [bx]", "mov ax, [ax:bx]",
		"mov ax, dword [bx]", "mov [bx], [di]", "mov [bx], 1", "mov byte [bx], -129",
		"mov word [bx], -32769", "mov word [bx], 65536", "inc [bx]", "push 1", "pop cs",
		"call byte [bx]", "jz ax", "shl ax, 2", "shl ax, dl", "imul ax, bx", "rep nop",
		"repnz movsb", "xchg ax, 1", "lea al, [bx]", "jz missing"} {
		t.Run(text, func(t *testing.T) {
			c := newCodec(t)
			nodes, err := c.Parse(t.Context(), strings.NewReader(".segment \"CODE\"\n"+text))
			if err == nil {
				_, err = c.Assemble(t.Context(), nodes)
			}
			assert.Error(t, err)
		})
	}
	_, err := asmx86.NewInstruction(isa.MovName, ast.NewInstructionArgument(isa.RegAX), ast.NewInstructionArgument(isa.RegFlags))
	assert.Error(t, err)
}

func checkNASM(t *testing.T, source, reference string) {
	t.Helper()
	if _, err := exec.LookPath("nasm"); err != nil {
		switch os.Getenv("RETROASM_REQUIRE_NASM") {
		case "1":
			t.Fatal(err)
		default:
			t.Skip("NASM is required for independent encoding checks")
		}
	}
	c := newCodec(t)
	nodes, err := c.Parse(t.Context(), strings.NewReader(".segment \"CODE\"\n"+source))
	assert.NoError(t, err)
	assembly, err := c.Assemble(t.Context(), nodes)
	assert.NoError(t, err)
	folder := t.TempDir()
	input := filepath.Join(folder, "probe.asm")
	output := filepath.Join(folder, "probe.bin")
	assert.NoError(t, os.WriteFile(input, []byte("cpu 8086\nbits 16\norg 0x100\n"+reference), 0600))
	diagnostic, err := exec.CommandContext(t.Context(), "nasm", "-O0", "-f", "bin", "-o", output, input).CombinedOutput()
	assert.NoError(t, err, "%s", diagnostic)
	want, err := os.ReadFile(output)
	assert.NoError(t, err)
	checkEncodingBytes(t, want, assembly.Binary)
}

func checkEncodingBytes(t *testing.T, want, got []byte) {
	t.Helper()
	first := 0
	for first < min(len(want), len(got)) && want[first] == got[first] {
		first++
	}
	assert.Len(t, got, len(want))
	assert.Equal(t, want[first:min(first+16, len(want))], got[first:min(first+16, len(got))], "first byte difference at %d", first)
}

func TestIndirectCallRetainsAddressing(t *testing.T) {
	for _, test := range []struct {
		operand ast.Node
		mode    isa.AddressingMode
	}{
		{operand: ast.NewInstructionArgument(isa.RegBX), mode: isa.RegisterAddressing},
		{operand: ast.NewInstructionArgument(parser.MemoryOperand{Address: isa.RegBXRef}), mode: isa.ModRMMemoryAddressing},
		{operand: ast.NewLabel("target"), mode: isa.RelativeAddressing},
	} {
		instruction, err := asmx86.NewInstruction(isa.CallName, test.operand)
		assert.NoError(t, err)
		assert.Equal(t, int(test.mode), instruction.Addressing)
	}
}
