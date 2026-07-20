package assembler

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retrogolib/assert"
)

var asm6EquTestCode = `
.segment "HEADER"

one EQU 1
plus EQU +
DB one plus one ;DB 1 + 1
`

func TestAssemblerAsm6EQU(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6EquTestCode)
	assert.NoError(t, err)
	assert.Equal(t, []byte{2}, b)
	assert.Len(t, b, 1)
}

var asm6AssignTestCode = `
.segment "HEADER"

i=1
.db i
j EQU i+1
k=i+1   ;k=1+1
.db k
i=j+1   ;i=i+1+1
.db i
i=k+1   ;i=2+1
.db i
`

func TestAssemblerAsm6Assign(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6AssignTestCode)
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 3}, b)
}

var asm6ImmediateConstantTestCode = `
.segment "HEADER"

i=32

LDA #i
`

func TestAssemblerAsm6ImmediateConstant(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6ImmediateConstantTestCode)
	assert.NoError(t, err)
	// LDA immediate = 0xA9, value = 32 (0x20)
	assert.Equal(t, []byte{0xA9, 0x20}, b)
}

var asm6ImmediateConstantUnderscoreTestCode = `
.segment "HEADER"

MAX_ENTITIES = 32
SPRITE_MASK = $FF

LDA #MAX_ENTITIES
ORA #SPRITE_MASK
`

func TestAssemblerAsm6ImmediateConstantUnderscore(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6ImmediateConstantUnderscoreTestCode)
	assert.NoError(t, err)
	// LDA immediate = 0xA9, value = 32 (0x20)
	// ORA immediate = 0x09, value = 255 (0xFF)
	assert.Equal(t, []byte{0xA9, 0x20, 0x09, 0xFF}, b)
}

var asm6ImmediateConstantExpressionTests = []struct {
	name string
	code string
	want []byte
}{
	{
		name: "identifier subtraction",
		code: "BOARD_WIDTH = 10\nLDY #(BOARD_WIDTH - 1)\n",
		want: []byte{0xA0, 0x09},
	},
	{
		name: "nested arithmetic",
		code: "LDY #((2 + 3) * 2)\n",
		want: []byte{0xA0, 0x0A},
	},
	{
		name: "parenthesized literal",
		code: "LDY #($0A)\n",
		want: []byte{0xA0, 0x0A},
	},
}

func TestAssemblerAsm6ImmediateConstantExpression(t *testing.T) {
	for _, test := range asm6ImmediateConstantExpressionTests {
		t.Run(test.name, func(t *testing.T) {
			code := ".segment \"HEADER\"\n" + test.code
			got, err := runAsm6Test(t, unitTestConfig, code)

			assert.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestAssemblerAsm6ImmediateConstantExpressionErrors(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
	}{
		{
			name:    "missing closing parenthesis",
			code:    "LDY #(10 - 1\n",
			message: "unexpected end of immediate expression",
		},
		{
			name:    "invalid token",
			code:    "LDY #(10, 1)\n",
			message: "unexpected token ',' in immediate expression",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := runAsm6Test(t, unitTestConfig, test.code)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.message)
		})
	}
}

var asm6IncbinTestCode = `
.segment "HEADER"
.incbin "test.bin"
`

func TestAssemblerAsm6Incbin(t *testing.T) {
	cfg := m6502.New()
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	reader := strings.NewReader(asm6IncbinTestCode)
	var buf bytes.Buffer
	asm := New(cfg, &buf)

	asm.fileReader = func(name string) ([]byte, error) {
		assert.Equal(t, "test.bin", name)
		return []byte{0xfe, 0xff}, nil
	}

	assert.NoError(t, asm.Process(t.Context(), reader))
	b := buf.Bytes()
	assert.Equal(t, []byte{0xfe, 0xff}, b)
}

var asm6DataModifierTestCode = `
.segment "HEADER"
DB "ABC"+1
DB "ABC"-"A"+32
`

func TestAssemblerAsm6DataModifier(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6DataModifierTestCode)
	assert.NoError(t, err)
	assert.Equal(t, []byte{'B', 'C', 'D', 32, 33, 34}, b)
}

var asm6AddressTestCode = `
.segment "HEADER"
DB 2
label:
< label, label
> label
label2:
DB 3
DL label2
DH label2
`

func TestAssemblerAsm6Address(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6AddressTestCode)
	assert.NoError(t, err)
	assert.Equal(t, []byte{2, 1, 1, 0, 3, 4, 0}, b)
}

var asm6HexTestCode = `
.segment "HEADER"
HEX 456789ABCDEF
HEX 0 1 23 4567
`

func TestAssemblerAsm6Hex(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6HexTestCode)
	assert.NoError(t, err)
	expected := []byte{
		0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
		0x00, 0x01, 0x23, 0x45, 0x67,
	}
	assert.Equal(t, expected, b)
}

var asm6DsbTestCode = `
.segment "HEADER"
space=3
DSB space,0x12
DSB 4
DSB 8,1
DSW 4,$ABCD
`

func TestAssemblerAsm6Dsb(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6DsbTestCode)
	assert.NoError(t, err)
	expected := []byte{
		0x12, 0x12, 0x12,
		0x00, 0x00, 0x00, 0x00,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0xcd, 0xab, 0xcd, 0xab, 0xcd, 0xab, 0xcd, 0xab,
	}
	assert.Equal(t, expected, b)
}

var asm6CurrentProgramAddressTestCode = `
.segment "HEADER"
DB 1,2,3
DSB $10-$
DSB $20-$, 1
$=$22
DSB $25-$, 2
`

func TestAssemblerAsm6CurrentProgramAddress(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6CurrentProgramAddressTestCode)
	assert.NoError(t, err)
	expected := []byte{
		1, 2, 3, // 3 items
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 13 items
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 16 items
		0, 0, // 2 items
		2, 2, 2, // 3 items
	}
	assert.Equal(t, expected, b)
}

var asm6PadTestCode = `
.segment "HEADER"
DB 1,2,3
PAD $10
PAD $20, 1
`

func TestAssemblerAsm6Pad(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6PadTestCode)
	assert.NoError(t, err)
	expected := []byte{
		1, 2, 3, // 3 items
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 13 items
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 16 items
	}
	assert.Equal(t, expected, b)
}

var asm6OrgTestCode = `
.segment "HEADER"
ORG $10
DB 1,2,3
ORG $20, 4
`

func TestAssemblerAsm6Org(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6OrgTestCode)
	assert.NoError(t, err)
	expected := []byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 16 items
		1, 2, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // 16 items
	}
	assert.Equal(t, expected, b)
}

var asm6AlignTestCode = `
.segment "HEADER"
DB 1,2,3
ALIGN 4
ALIGN 8,$EA
`

func TestAssemblerAsm6Align(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6AlignTestCode)
	assert.NoError(t, err)
	expected := []byte{
		1, 2, 3, 0, 0xea, 0xea, 0xea, 0xea, // 8 items
	}
	assert.Equal(t, expected, b)
}

var asm6AlignAlreadyAlignedCode = `
.segment "HEADER"
DB 1,2,3,4
ALIGN 4
DB $FF
`

func TestAssemblerAsm6AlignAlreadyAligned(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6AlignAlreadyAlignedCode)
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4, 0xff}, b)
}

var asm6ForwardRefAbsoluteCode = `
.segment "HEADER"
LDA forward,X
NOP
forward:
DB $42
`

func TestAssemblerAsm6ForwardRefAbsoluteAddressing(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6ForwardRefAbsoluteCode)
	assert.NoError(t, err)

	// Forward references must reserve absolute-width encoding during address assignment.
	expected := []byte{
		0xbd, 0x04, 0x00,
		0xea,
		0x42,
	}
	assert.Equal(t, expected, b)
}

var asm6FillValueTestCode = `
.segment "HEADER"
FILLVALUE $FF
DB 0
ALIGN 4
FILLVALUE $FF-1
PAD 8
`

func TestAssemblerAsm6FillValue(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6FillValueTestCode)
	assert.NoError(t, err)
	expected := []byte{
		0x00, 0xff, 0xff, 0xff, 0xfe, 0xfe, 0xfe, 0xfe, // 8 items
	}
	assert.Equal(t, expected, b)
}

var asm6BaseTestCode = `
.segment "HEADER"
BASE $10
oldaddr=$
PAD $15, 0x12
PAD oldaddr+9, 0x34
`

func TestAssemblerAsm6Base(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6BaseTestCode)
	assert.NoError(t, err)
	expected := []byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 16 items
		0x12, 0x12, 0x12, 0x12, 0x12, // 5 items
		0x34, 0x34, 0x34, 0x34, // 4 items
	}
	assert.Equal(t, expected, b)
}

var asm6IfEndifTestCode = `
.segment "HEADER"
i=1
j=0

IF j>0
	DB i/j
ELSEIF i<2
	DB 1
ELSE
	DB 2
ENDIF

IF i<=1
	DB 3
ENDIF
IF i==0
	DB 3
ENDIF
`

var asm6ElseOutsideOfContext = `
.segment "HEADER"
ELSE
`

var asm6MultipleElse = `
.segment "HEADER"
i = 1
IF i > 0
ELSE
ELSE
`

var asm6EndifOutsideOfContext = `
.segment "HEADER"
ENDIF
`

var asm6MissingEndif = `
.segment "HEADER"
i = 1
IF i > 0
`

var asm6ElseifOutsideOfContext = `
.segment "HEADER"
ELSEIF 1 > 0
`

var asm6ElseifReferenceProgramCounter = `
.segment "HEADER"
IF 1 > 0
ELSEIF $ > 0
ENDIF
`

var asm6IfReferenceProgramCounter = `
.segment "HEADER"
IF $ > 0
ENDIF
`

func TestAssemblerAsm6IfElseElseIfEndif(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6IfEndifTestCode)
	assert.NoError(t, err)
	expected := []byte{
		0x1, // 1 item
		0x3, // 1 item
	}
	assert.Equal(t, expected, b)

	_, err = runAsm6Test(t, unitTestConfig, asm6ElseOutsideOfContext)
	assert.ErrorIs(t, err, errConditionOutsideIfContext)

	_, err = runAsm6Test(t, unitTestConfig, asm6MultipleElse)
	assert.ErrorIs(t, err, errMultipleElseFound)

	_, err = runAsm6Test(t, unitTestConfig, asm6EndifOutsideOfContext)
	assert.ErrorIs(t, err, errConditionOutsideIfContext)

	_, err = runAsm6Test(t, unitTestConfig, asm6MissingEndif)
	assert.ErrorIs(t, err, errMissingEndif)

	_, err = runAsm6Test(t, unitTestConfig, asm6ElseifOutsideOfContext)
	assert.ErrorIs(t, err, errConditionOutsideIfContext)

	_, err = runAsm6Test(t, unitTestConfig, asm6ElseifReferenceProgramCounter)
	assert.ErrorIs(t, err, errExpressionCantReferenceProgramCounter)

	_, err = runAsm6Test(t, unitTestConfig, asm6IfReferenceProgramCounter)
	assert.ErrorIs(t, err, errExpressionCantReferenceProgramCounter)
}

var asm6IfIfdefTestCode = `
.segment "HEADER"
i=1

IFDEF i
	DB 1
ELSE
	DB 2
ENDIF

IFNDEF j
	DB 3
ELSE
	DB 4
ENDIF

IFDEF k
	DB 6
ENDIF
`

func TestAssemblerAsm6IfdefIfndef(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6IfIfdefTestCode)
	assert.NoError(t, err)
	expected := []byte{
		0x1, // 1 item
		0x3, // 1 item
	}
	assert.Equal(t, expected, b)
}

var asm6MacroCode = `
.segment "HEADER"

MACRO setAXY x,y,z
	LDA #x
	LDX #y
	LDY #z
ENDM

setAXY $12,$34,$56
`

func TestAssemblerAsm6Macro(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6MacroCode)
	assert.NoError(t, err)
	expected := []byte{
		0xa9, 0x12, // 2 items
		0xa2, 0x34, // 2 items
		0xa0, 0x56, // 2 items
	}
	assert.Equal(t, expected, b)
}

var asm6ErrorCode = `
.segment "HEADER"
x=101
IF x<100
	ERROR "should not trigger error"
ENDIF
IF x>100
	ERROR "X is out of range :("
ENDIF
`

func TestAssemblerAsm6Error(t *testing.T) {
	_, err := runAsm6Test(t, unitTestConfig, asm6ErrorCode)
	assert.ErrorContains(t, err, "X is out of range :(")
}

var asm6EnumCode = `
.segment "HEADER"

BASE $2
db 3

ENUM $0
	foo:    db 1
	foo2:   db 2
ENDE
`

func TestAssemblerAsm6Enum(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6EnumCode)
	assert.NoError(t, err)
	expected := []byte{
		0x01, // 1 item
		0x02, // 1 item
		0x03, // 1 item
	}
	assert.Equal(t, expected, b)
}

var asm6ReptCode = `
.segment "HEADER"

i=0
REPT 3
	DB i
	i=i+1
ENDR
DB 0xff
`

var asm6ReptReferenceProgramCounter = `
.segment "HEADER"
REPT $
ENDR
`

var asm6Rept0 = `
.segment "HEADER"
REPT 0
  DB 1
ENDR
`

var asm6Rept0Eval = `
.segment "HEADER"
i = 0
REPT i
  DB 1
ENDR
`

func TestAssemblerAsm6Rept(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6ReptCode)
	assert.NoError(t, err)
	expected := []byte{
		0x00, // 1 item
		0x01, // 1 item
		0x02, // 1 item
		0xff, // 1 item
	}
	assert.Equal(t, expected, b)

	b, err = runAsm6Test(t, unitTestConfig, asm6Rept0)
	assert.NoError(t, err)
	assert.Empty(t, b, "expected no output")

	b, err = runAsm6Test(t, unitTestConfig, asm6Rept0Eval)
	assert.NoError(t, err)
	assert.Empty(t, b, "expected no output")

	_, err = runAsm6Test(t, unitTestConfig, asm6ReptReferenceProgramCounter)
	assert.ErrorIs(t, err, errExpressionCantReferenceProgramCounter)
}

var asm6IdentifierTestCode = `
.segment "HEADER"

PPU_ADDR = $2006
PPU_STATUS = $2002
temp_val = $20

; Test identifiers as instruction arguments
lda PPU_STATUS    ; Absolute addressing with identifier
sta PPU_ADDR      ; Absolute addressing with identifier
lda temp_val      ; Zero page addressing (identifiers optimize to zero page when value fits)
sta temp_val      ; Zero page addressing (identifiers optimize to zero page when value fits)
`

func TestAssemblerAsm6Identifier(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6IdentifierTestCode)
	assert.NoError(t, err)
	// LDA $2002 (absolute) = AD 02 20
	// STA $2006 (absolute) = 8D 06 20
	// LDA $20 (zero page) = A5 20
	// STA $20 (zero page) = 85 20
	expected := []byte{0xAD, 0x02, 0x20, 0x8D, 0x06, 0x20, 0xA5, 0x20, 0x85, 0x20}
	assert.Equal(t, expected, b)
}

var asm6BranchIdentifierTestCode = `
.segment "HEADER"

loop:
  lda #$01
  beq done
  bne loop
done:
  brk
`

func TestAssemblerAsm6BranchIdentifier(t *testing.T) {
	b, err := runAsm6Test(t, unitTestConfig, asm6BranchIdentifierTestCode)
	assert.NoError(t, err)
	// LDA #$01 (at 0x0000) = A9 01
	// BEQ done (at 0x0002, target 0x0006, offset +2) = F0 02
	// BNE loop (at 0x0004, target 0x0000, offset -6) = D0 FA
	// BRK (at 0x0006) = 00
	expected := []byte{0xA9, 0x01, 0xF0, 0x02, 0xD0, 0xFA, 0x00}
	assert.Equal(t, expected, b)
}

var asm6BranchOutOfRangeTestCode = `
.segment "HEADER"

start:
.org $0000
  lda #$01
far_away:
.org $0100
  beq start    ; This should fail - too far
`

func TestAssemblerAsm6BranchOutOfRange(t *testing.T) {
	_, err := runAsm6Test(t, unitTestConfig, asm6BranchOutOfRangeTestCode)
	assert.NotNil(t, err, "expected error for out of range branch")
	// Verify the error message contains useful debugging info
	assert.Contains(t, err.Error(), "branch target")
	assert.Contains(t, err.Error(), "too far")
}

func TestAssemblerContextCancellation(t *testing.T) {
	cfg := m6502.New()
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	// Test cancellation during parsing
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	reader := strings.NewReader(asm6EquTestCode)
	var buf bytes.Buffer
	asm := New(cfg, &buf)

	err := asm.Process(ctx, reader)
	assert.NotNil(t, err, "expected error from cancelled context")
	assert.ErrorIs(t, err, context.Canceled, "expected context.Canceled error")
}

var asm6SourceIncludeTestCode = `
.segment "HEADER"
.include "defs.asm"
DB myval
`

func TestAssemblerAsm6SourceInclude(t *testing.T) {
	cfg := m6502.New()
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	reader := strings.NewReader(asm6SourceIncludeTestCode)
	var buf bytes.Buffer
	asm := New(cfg, &buf)

	asm.fileReader = func(name string) ([]byte, error) {
		assert.Equal(t, "defs.asm", name)
		return []byte("myval = $42\n"), nil
	}

	assert.NoError(t, asm.Process(t.Context(), reader))
	assert.Equal(t, []byte{0x42}, buf.Bytes())
}

func TestAssemblerAsm6SourceIncludeCycle(t *testing.T) {
	cfg := m6502.New()
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))

	reader := strings.NewReader(asm6SourceIncludeTestCode)
	var buf bytes.Buffer
	asm := New(cfg, &buf)

	asm.fileReader = func(name string) ([]byte, error) {
		switch name {
		case "defs.asm":
			return []byte(".include \"more.asm\"\n"), nil
		case "more.asm":
			return []byte(".include \"defs.asm\"\n"), nil
		default:
			return nil, fmt.Errorf("unexpected include %q", name)
		}
	}

	err := asm.Process(t.Context(), reader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "include cycle detected")
	assert.Contains(t, err.Error(), "defs.asm -> more.asm -> defs.asm")
}

func runAsm6Test(t *testing.T, testConfig, testCode string) ([]byte, error) {
	t.Helper()

	cfg := m6502.New()
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(testConfig)))

	reader := strings.NewReader(testCode)
	var buf bytes.Buffer
	asm := New(cfg, &buf)

	err := asm.Process(t.Context(), reader)
	b := buf.Bytes()
	return b, err
}
