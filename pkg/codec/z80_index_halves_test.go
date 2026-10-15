package codec_test

import (
	"fmt"
	"strings"
	"testing"

	asmz80 "github.com/retroenv/retroasm/pkg/arch/z80"
	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	z80profile "github.com/retroenv/retroasm/pkg/arch/z80/profile"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

type z80IndexHalfCase struct {
	mnemonic string
	operands z80parser.Operands
	encoded  []byte
}

func TestZ80IndexHalvesRoundTrip(t *testing.T) {
	for _, test := range z80IndexHalfCases() {
		t.Run(fmt.Sprintf("%02x/%02x", test.encoded[0], test.encoded[1]), func(t *testing.T) {
			testZ80IndexHalfRoundTrip(t, test)
		})
	}
}

func TestZ80IndexHalvesRejectIncompatibleRegisters(t *testing.T) {
	c := newZ80Codec(t)
	for _, half := range []z80.RegisterParam{z80.RegIXH, z80.RegIXL, z80.RegIYH, z80.RegIYL} {
		for _, other := range []z80.RegisterParam{z80.RegH, z80.RegL, z80.RegIXH, z80.RegIXL, z80.RegIYH, z80.RegIYL} {
			if strings.HasPrefix(half.String(), "ix") == strings.HasPrefix(other.String(), "ix") &&
				other != z80.RegH && other != z80.RegL {

				continue
			}
			for _, registers := range [][2]z80.RegisterParam{{half, other}, {other, half}} {
				name := fmt.Sprintf("ld %s,%s", registers[0], registers[1])
				t.Run(name, func(t *testing.T) {
					_, err := codec.BuildInstruction(c, z80.LdName, z80parser.Operands{
						z80parser.RegisterOperand(registers[0]),
						z80parser.RegisterOperand(registers[1]),
					})
					assert.Error(t, err)
					_, err = c.ParseInstruction(t.Context(), strings.NewReader(name))
					assert.Error(t, err)
				})
			}
		}
	}
}

func TestZ80IndexHalfMetadataRetainsIndexedImmediateStores(t *testing.T) {
	c := newZ80Codec(t)
	for index, register := range []z80.RegisterParam{z80.RegIX, z80.RegIY} {
		for _, offset := range []int64{-128, -1, 0, 127} {
			for _, value := range []uint64{0, 0x42, 0xff} {
				t.Run(fmt.Sprintf("%s/%d/%d", register, offset, value), func(t *testing.T) {
					displacement := uint64(uint8(offset))
					built, err := codec.BuildInstruction(c, z80.LdName, z80parser.Operands{
						z80parser.IndexedOperand(register, ast.NewNumber(displacement)),
						z80parser.ValueOperand(ast.NewNumber(value)),
					})
					assert.NoError(t, err)
					formatted, err := c.FormatInstruction(built)
					assert.NoError(t, err)
					parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
					assert.NoError(t, err)
					for _, instruction := range []ast.Instruction{built, parsed} {
						result, err := c.Assemble(t.Context(), []ast.Node{instruction})
						assert.NoError(t, err)
						assert.Equal(t, []byte{[]byte{z80.PrefixDD, z80.PrefixFD}[index], 0x36, byte(offset), byte(value)},
							result.Binary)
					}
				})
			}
		}
	}
}

func testZ80IndexHalfRoundTrip(t *testing.T, test z80IndexHalfCase) {
	t.Helper()
	c := newZ80Codec(t)
	built, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
	assert.NoError(t, err)
	formatted, err := c.FormatInstruction(built)
	assert.NoError(t, err)
	parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
	assert.NoError(t, err)
	for _, instruction := range []ast.Instruction{built, parsed} {
		assert.NoError(t, c.ValidateInstruction(instruction))
		result, err := c.Assemble(t.Context(), []ast.Node{instruction})
		assert.NoError(t, err)
		assert.Equal(t, test.encoded, result.Binary)
	}
	for _, profile := range []z80profile.Kind{z80profile.StrictDocumented, z80profile.GameBoySubset} {
		strict := newZ80Codec(t, asmz80.WithProfile(profile))
		_, err := codec.BuildInstruction(strict, test.mnemonic, test.operands)
		assert.Error(t, err)
		_, err = strict.ParseInstruction(t.Context(), strings.NewReader(formatted))
		assert.Error(t, err)
	}
	memory := z80.NewBasicMemory()
	for index, value := range test.encoded {
		memory.Write(uint16(index), value)
	}
	cpu, err := z80.New(memory)
	assert.NoError(t, err)
	before := cpu.Cycles()
	assert.NoError(t, cpu.Step())
	assert.Equal(t, uint16(len(test.encoded)), cpu.PC)
	cycles := uint64(8)
	if len(test.encoded) == 3 {
		cycles = 11
	}
	assert.Equal(t, before+cycles, cpu.Cycles())
}

func z80IndexHalfCases() []z80IndexHalfCase {
	cases := make([]z80IndexHalfCase, 0, 92)
	for _, prefix := range []byte{z80.PrefixDD, z80.PrefixFD} {
		high, low := z80.RegIXH, z80.RegIXL
		if prefix == z80.PrefixFD {
			high, low = z80.RegIYH, z80.RegIYL
		}
		registers := []z80.RegisterParam{z80.RegB, z80.RegC, z80.RegD, z80.RegE, high, low, z80.RegNone, z80.RegA}
		for _, lane := range []byte{4, 5} {
			for index, mnemonic := range []string{z80.IncName, z80.DecName, z80.LdName} {
				operands := z80parser.Operands{z80parser.RegisterOperand(registers[lane])}
				encoded := []byte{prefix, 0x04 + lane*8 + byte(index)}
				if mnemonic == z80.LdName {
					operands = append(operands, z80parser.ValueOperand(ast.NewNumber(0x37)))
					encoded = append(encoded, 0x37)
				}
				cases = append(cases, z80IndexHalfCase{
					mnemonic: mnemonic,
					operands: operands,
					encoded:  encoded,
				})
			}
			for index, mnemonic := range []string{z80.AddName, z80.AdcName, z80.SubName, z80.SbcName,
				z80.AndName, z80.XorName, z80.OrName, z80.CpName} {
				operands := z80parser.Operands{z80parser.RegisterOperand(registers[lane])}
				if mnemonic == z80.AddName || mnemonic == z80.AdcName || mnemonic == z80.SbcName {
					operands = append(z80parser.Operands{z80parser.RegisterOperand(z80.RegA)}, operands...)
				}
				cases = append(cases, z80IndexHalfCase{
					mnemonic: mnemonic,
					operands: operands,
					encoded:  []byte{prefix, 0x80 + byte(index)*8 + lane},
				})
			}
		}
		cases = append(cases, z80IndexHalfCopyCases(prefix, registers)...)
	}
	return cases
}

func z80IndexHalfCopyCases(prefix byte, registers []z80.RegisterParam) []z80IndexHalfCase {
	var cases []z80IndexHalfCase
	for dst, destination := range registers {
		for src, source := range registers {
			if destination == z80.RegNone || source == z80.RegNone || (dst != 4 && dst != 5 && src != 4 && src != 5) {
				continue
			}
			cases = append(cases, z80IndexHalfCase{
				mnemonic: z80.LdName,
				operands: z80parser.Operands{z80parser.RegisterOperand(destination), z80parser.RegisterOperand(source)},
				encoded:  []byte{prefix, byte(0x40 + dst*8 + src)},
			})
		}
	}
	return cases
}
