package x86_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	asmx86 "github.com/retroenv/retroasm/pkg/arch/x86"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
	"github.com/retroenv/retrogolib/assert"
)

func newCodec(t *testing.T) *codec.Codec[*isa.Instruction] {
	t.Helper()
	configuration := asmx86.New()
	segment := &config.Segment{
		Memory: config.Memory{
			Name:  "CODE",
			Start: 0x100,
			Size:  0xff00,
		},
		SegmentName:  "CODE",
		SegmentStart: 0x100,
	}
	configuration.Segments = map[string]*config.Segment{"CODE": segment}
	configuration.SegmentsOrdered = []*config.Segment{segment}
	result, err := codec.New(configuration)
	assert.NoError(t, err)
	return result
}

func TestTypedExitProbe(t *testing.T) {
	c := newCodec(t)
	mov, err := asmx86.NewInstruction(isa.MovName, ast.NewInstructionArgument(isa.RegAX), ast.NewNumber(0x4c2a))
	assert.NoError(t, err)
	interrupt, err := asmx86.NewInstruction(isa.IntName, ast.NewNumber(0x21))
	assert.NoError(t, err)
	assert.Equal(t, int(isa.ImmediateAddressing), mov.Addressing)
	assert.Equal(t, int(isa.ImmediateAddressing), interrupt.Addressing)
	assert.True(t, mov.OpcodeID.ValidFor(arch.X86))
	assert.NotEqual(t, mov.OpcodeID, interrupt.OpcodeID)
	assembly, err := c.Assemble(t.Context(), []ast.Node{ast.NewSegment("CODE"), mov, interrupt})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xb8, 0x2a, 0x4c, 0xcd, 0x21}, assembly.Binary)
}

func TestEncodingMatchesNASM(t *testing.T) {
	if _, err := exec.LookPath("nasm"); err != nil {
		t.Skip("NASM is required for independent encoding checks")
	}
	var source strings.Builder
	source.WriteString("cld\n")
	for destination := isa.RegAL; destination <= isa.RegDI; destination++ {
		immediate := "0xab"
		if destination.Is16Bit() {
			immediate = "0xabcd"
		}
		source.WriteString("mov " + destination.String() + ", " + immediate + "\n")
		for origin := isa.RegAL; origin <= isa.RegDI; origin++ {
			if origin.GetRegisterSize() == destination.GetRegisterSize() {
				source.WriteString("mov " + destination.String() + ", " + origin.String() + "\n")
			}
		}
	}

	appendArithmeticForms(&source)
	for register := isa.RegAX; register <= isa.RegDI; register++ {
		source.WriteString("push " + register.String() + "\npop " + register.String() + "\n")
	}
	for register := isa.RegES; register <= isa.RegDS; register++ {
		if register != isa.RegCS {
			source.WriteString("mov " + register.String() + ", ax\n")
		}
		source.WriteString("mov ax, " + register.String() + "\n")
	}
	source.WriteString("mov al, 0\nmov ah, 255\nmov ax, 256\nmov sp, 65535\n")
	source.WriteString("mov ds, ax\nmov ax, cs\npush bp\npop bp\ncall forward\nbackward:\nnop\nforward:\ncall backward\njmp backward\nmov ah, 0x4c\nint 0x21\nret\n")
	c := newCodec(t)
	nodes, err := c.Parse(t.Context(), strings.NewReader(".segment \"CODE\"\n"+source.String()))
	assert.NoError(t, err)
	assembly, err := c.Assemble(t.Context(), nodes)
	assert.NoError(t, err)
	folder := t.TempDir()
	asmPath := filepath.Join(folder, "probe.asm")
	binPath := filepath.Join(folder, "probe.com")
	assert.NoError(t, os.WriteFile(asmPath, []byte("cpu 8086\nbits 16\norg 0x100\n"+strings.ReplaceAll(source.String(), "jmp backward", "jmp near backward")), 0600))
	output, err := exec.CommandContext(t.Context(), "nasm", "-O0", "-f", "bin", "-o", binPath, asmPath).CombinedOutput()
	assert.NoError(t, err, "%s", output)
	expected, err := os.ReadFile(binPath)
	assert.NoError(t, err)
	assert.Equal(t, expected, assembly.Binary)
}

func TestRejectsUnsupportedForms(t *testing.T) {
	for _, text := range []string{
		"mov al, 256", "mov ax, 65536", "mov ax, bl", "mov cs, ax", "mov ds, 1",
		"int 256", "push al", "ret 2", "add ds, ax", "mov ax", "mov ax, [bx]", "bound ax, bx",
	} {
		t.Run(text, func(t *testing.T) {
			c := newCodec(t)
			nodes, err := c.Parse(t.Context(), strings.NewReader(".segment \"CODE\"\n"+text))
			if err == nil {
				_, err = c.Assemble(t.Context(), nodes)
			}
			assert.Error(t, err)
		})
	}
	_, err := asmx86.NewInstruction("unknown")
	assert.Error(t, err)
}

func TestConditionalBranchBoundaries(t *testing.T) {
	for _, test := range []struct {
		name         string
		displacement int
		expected     byte
		valid        bool
	}{
		{
			name:         "negative limit",
			displacement: -128,
			expected:     0x80,
			valid:        true,
		},
		{
			name:         "positive limit",
			displacement: 127,
			expected:     0x7f,
			valid:        true,
		},
		{
			name:         "negative overflow",
			displacement: -129,
		},
		{
			name:         "positive overflow",
			displacement: 128,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := newCodec(t)
			instruction, err := asmx86.NewInstruction(isa.JzName, ast.NewNumber(uint64(0x102+test.displacement)))
			assert.NoError(t, err)
			assembly, err := c.Assemble(t.Context(), []ast.Node{ast.NewSegment("CODE"), instruction})
			if !test.valid {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, []byte{0x74, test.expected}, assembly.Binary)
		})
	}
}

func TestFormatRetainsOperands(t *testing.T) {
	c := newCodec(t)
	for _, source := range []string{"mov ax, 0xFFFF", "mov al, ah", "mov ds, ax", "call target", "int 0x21", "cld"} {
		first, err := c.ParseInstruction(t.Context(), strings.NewReader(source))
		assert.NoError(t, err)
		text, err := c.FormatInstruction(first)
		assert.NoError(t, err)
		second, err := c.ParseInstruction(t.Context(), strings.NewReader(text))
		assert.NoError(t, err)
		assert.Equal(t, first.OpcodeID, second.OpcodeID)
		assert.Equal(t, source, text)
	}
}

func appendArithmeticForms(source *strings.Builder) {
	for _, mnemonic := range []string{isa.AddName, isa.SubName, isa.CmpName, isa.AndName, isa.OrName, isa.XorName} {
		for destination := isa.RegAL; destination <= isa.RegDI; destination++ {
			for origin := isa.RegAL; origin <= isa.RegDI; origin++ {
				if origin.GetRegisterSize() == destination.GetRegisterSize() {
					source.WriteString(mnemonic + " " + destination.String() + ", " + origin.String() + "\n")
				}
			}
			if destination != isa.RegAL && destination != isa.RegAX {
				immediate := "0xab"
				if destination.Is16Bit() {
					immediate = "0xabcd"
				}
				source.WriteString(mnemonic + " " + destination.String() + ", " + immediate + "\n")
			}
		}
	}
}
