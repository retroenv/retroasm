package assembler

import (
	"bytes"
	"strings"
	"testing"

	"github.com/retroenv/assembler/arch"
	"github.com/retroenv/assembler/assembler/config"
	"github.com/retroenv/retrogolib/assert"
)

var asm6EquTestCode = `
.segment "HEADER"

one EQU 1
plus EQU +
DB one plus one ;DB 1 + 1
`

func TestAssemblerAsm6EQU(t *testing.T) {
	b := runAsm6Test(t, unitTestConfig, asm6EquTestCode)
	assert.Equal(t, []byte{2}, b)
	assert.Equal(t, 1, len(b))
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
	b := runAsm6Test(t, unitTestConfig, asm6AssignTestCode)
	assert.Equal(t, []byte{1, 2, 3, 3}, b)
}

var asm6IncbinTestCode = `
.segment "HEADER"
.incbin "test.bin"
`

func TestAssemblerAsm6Incbin(t *testing.T) {
	cfg := &config.Config{}
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(unitTestConfig)))
	cfg.Arch = arch.NewNES()

	reader := strings.NewReader(asm6IncbinTestCode)
	var buf bytes.Buffer
	asm := New(cfg, reader, &buf)

	asm.fileReader = func(name string) ([]byte, error) {
		assert.Equal(t, "test.bin", name)
		return []byte{0xfe, 0xff}, nil
	}

	assert.NoError(t, asm.Process())
	b := buf.Bytes()
	assert.Equal(t, []byte{0xfe, 0xff}, b)
}

var asm6DataModifierTestCode = `
.segment "HEADER"
DB "ABC"+1
DB "ABC"-"A"+32
`

func TestAssemblerAsm6DataModifier(t *testing.T) {
	b := runAsm6Test(t, unitTestConfig, asm6DataModifierTestCode)
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
	b := runAsm6Test(t, unitTestConfig, asm6AddressTestCode)
	assert.Equal(t, []byte{2, 1, 1, 0, 3, 4, 0}, b)
}

var asm6HexTestCode = `
.segment "HEADER"
HEX 456789ABCDEF
HEX 0 1 23 4567
`

func TestAssemblerAsm6Hex(t *testing.T) {
	b := runAsm6Test(t, unitTestConfig, asm6HexTestCode)
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
	b := runAsm6Test(t, unitTestConfig, asm6DsbTestCode)
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
$=$1000
DSB $1005-$, 2
`

func TestAssemblerAsm6CurrentProgramAddress(t *testing.T) {
	b := runAsm6Test(t, unitTestConfig, asm6CurrentProgramAddressTestCode)
	expected := []byte{
		1, 2, 3, // 3 items
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 13 items
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 16 items
		2, 2, 2, 2, 2, // 5 items
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
	b := runAsm6Test(t, unitTestConfig, asm6PadTestCode)
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
	b := runAsm6Test(t, unitTestConfig, asm6OrgTestCode)
	expected := []byte{
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
	b := runAsm6Test(t, unitTestConfig, asm6AlignTestCode)
	expected := []byte{
		1, 2, 3, 0, 0xea, 0xea, 0xea, 0xea, // 8 items
	}
	assert.Equal(t, expected, b)
}

var asm6FillValueTestCode = `
.segment "HEADER"
FILLVALUE $FF
ALIGN 4
FILLVALUE $FF-1
PAD 8
`

func TestAssemblerAsm6FillValue(t *testing.T) {
	b := runAsm6Test(t, unitTestConfig, asm6FillValueTestCode)
	expected := []byte{
		0xff, 0xff, 0xff, 0xff, 0xfe, 0xfe, 0xfe, 0xfe, // 8 items
	}
	assert.Equal(t, expected, b)
}

var asm6BaseTestCode = `
.segment "HEADER"
BASE $6000
oldaddr=$
PAD $6005, 0x12
PAD oldaddr+9, 0x34
`

func TestAssemblerAsm6Base(t *testing.T) {
	b := runAsm6Test(t, unitTestConfig, asm6BaseTestCode)
	expected := []byte{
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

func TestAssemblerAsm6IfElseElseIfEndif(t *testing.T) {
	b := runAsm6Test(t, unitTestConfig, asm6IfEndifTestCode)
	expected := []byte{
		0x1, // 1 item
		0x3, // 1 item
	}
	assert.Equal(t, expected, b)
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

`

func TestAssemblerAsm6IfdefIfndef(t *testing.T) {
	b := runAsm6Test(t, unitTestConfig, asm6IfIfdefTestCode)
	expected := []byte{
		0x1, // 1 item
		0x3, // 1 item
	}
	assert.Equal(t, expected, b)
}

func runAsm6Test(t *testing.T, testConfig, testCode string) []byte {
	t.Helper()

	cfg := &config.Config{}
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(testConfig)))
	cfg.Arch = arch.NewNES()

	reader := strings.NewReader(testCode)
	var buf bytes.Buffer
	asm := New(cfg, reader, &buf)

	assert.NoError(t, asm.Process())
	b := buf.Bytes()
	return b
}
