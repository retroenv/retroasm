package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

var parseIdentifierTests = []struct {
	name           string
	mnemonic       string
	tokens         []token.Token
	variants       []*cpuz80.Instruction
	wantVariant    *cpuz80.Instruction
	wantAddressing cpuz80.AddressingMode
	wantRegister   []cpuz80.RegisterParam
	wantValues     []ast.Node
}{
	{
		name:           "nop implied",
		mnemonic:       cpuz80.NopName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "nop"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.Nop},
		wantVariant:    cpuz80.Nop,
		wantAddressing: cpuz80.ImpliedAddressing,
	},
	{
		name:           "ret implied selects unconditional variant",
		mnemonic:       cpuz80.RetName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ret"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.RetCond, cpuz80.Ret},
		wantVariant:    cpuz80.Ret,
		wantAddressing: cpuz80.ImpliedAddressing,
	},
	{
		name:           "ld register pair",
		mnemonic:       cpuz80.LdName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.Identifier, Value: "b"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8, cpuz80.LdReg16},
		wantVariant:    cpuz80.LdReg8,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA, cpuz80.RegB},
	},
	{
		name:           "ld a immediate",
		mnemonic:       cpuz80.LdName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.Number, Value: "42"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8, cpuz80.LdReg16},
		wantVariant:    cpuz80.LdImm8,
		wantAddressing: cpuz80.ImmediateAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
		wantValues:     []ast.Node{ast.NewNumber(42)},
	},
	{
		name:           "ld hl immediate 16-bit value",
		mnemonic:       cpuz80.LdName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "hl"}, {Type: token.Comma}, {Type: token.Number, Value: "$1234"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8, cpuz80.LdReg16},
		wantVariant:    cpuz80.LdReg16,
		wantAddressing: cpuz80.ImmediateAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegHL},
		wantValues:     []ast.Node{ast.NewNumber(0x1234)},
	},
	{
		name:           "jr relative",
		mnemonic:       cpuz80.JrName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "jr"}, {Type: token.Identifier, Value: "loop"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.JrCond, cpuz80.JrRel},
		wantVariant:    cpuz80.JrRel,
		wantAddressing: cpuz80.RelativeAddressing,
		wantValues:     []ast.Node{ast.NewLabel("loop")},
	},
	{
		name:           "jr conditional relative",
		mnemonic:       cpuz80.JrName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "jr"}, {Type: token.Identifier, Value: "nz"}, {Type: token.Comma}, {Type: token.Identifier, Value: "loop"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.JrRel, cpuz80.JrCond},
		wantVariant:    cpuz80.JrCond,
		wantAddressing: cpuz80.RelativeAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegCondNZ},
		wantValues:     []ast.Node{ast.NewLabel("loop")},
	},
	{
		name:           "jp absolute",
		mnemonic:       cpuz80.JpName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "jp"}, {Type: token.Identifier, Value: "target"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.JpCond, cpuz80.JpAbs},
		wantVariant:    cpuz80.JpAbs,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantValues:     []ast.Node{ast.NewLabel("target")},
	},
	{
		name:           "jp absolute with tokenized plus offset",
		mnemonic:       cpuz80.JpName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "jp"}, {Type: token.Identifier, Value: "target"}, {Type: token.Plus}, {Type: token.Number, Value: "2"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.JpCond, cpuz80.JpAbs},
		wantVariant:    cpuz80.JpAbs,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantValues:     []ast.Node{expressionNode(token.Token{Type: token.Identifier, Value: "target"}, token.Token{Type: token.Plus}, token.Token{Type: token.Number, Value: "2"})},
	},
	{
		name:           "jp absolute with chained offsets",
		mnemonic:       cpuz80.JpName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "jp"}, {Type: token.Identifier, Value: "target"}, {Type: token.Plus}, {Type: token.Number, Value: "3"}, {Type: token.Minus}, {Type: token.Number, Value: "1"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.JpCond, cpuz80.JpAbs},
		wantVariant:    cpuz80.JpAbs,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantValues: []ast.Node{
			expressionNode(
				token.Token{Type: token.Identifier, Value: "target"},
				token.Token{Type: token.Plus},
				token.Token{Type: token.Number, Value: "3"},
				token.Token{Type: token.Minus},
				token.Token{Type: token.Number, Value: "1"},
			),
		},
	},
	{
		name:           "jp absolute with symbolic expression",
		mnemonic:       cpuz80.JpName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "jp"}, {Type: token.Identifier, Value: "target"}, {Type: token.Plus}, {Type: token.Identifier, Value: "delta"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.JpCond, cpuz80.JpAbs},
		wantVariant:    cpuz80.JpAbs,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantValues:     []ast.Node{expressionNode(token.Token{Type: token.Identifier, Value: "target"}, token.Token{Type: token.Plus}, token.Token{Type: token.Identifier, Value: "delta"})},
	},
	{
		name:           "jp conditional with c uses condition code",
		mnemonic:       cpuz80.JpName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "jp"}, {Type: token.Identifier, Value: "c"}, {Type: token.Comma}, {Type: token.Identifier, Value: "target"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.JpAbs, cpuz80.JpCond},
		wantVariant:    cpuz80.JpCond,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegCondC},
		wantValues:     []ast.Node{ast.NewLabel("target")},
	},
	{
		name:           "call absolute",
		mnemonic:       cpuz80.CallName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "call"}, {Type: token.Identifier, Value: "target"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.CallCond, cpuz80.Call},
		wantVariant:    cpuz80.Call,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantValues:     []ast.Node{ast.NewLabel("target")},
	},
	{
		name:     "ld a,(nn) extended load",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Number, Value: "$1234"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdExtended},
		wantVariant:    cpuz80.LdExtended,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegLoadExtA},
		wantValues:     []ast.Node{ast.NewNumber(0x1234)},
	},
	{
		name:     "ld a,(label+n) extended load",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "table"},
			{Type: token.Plus},
			{Type: token.Number, Value: "1"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdExtended},
		wantVariant:    cpuz80.LdExtended,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegLoadExtA},
		wantValues:     []ast.Node{expressionNode(token.Token{Type: token.Identifier, Value: "table"}, token.Token{Type: token.Plus}, token.Token{Type: token.Number, Value: "1"})},
	},
	{
		name:     "ld a,(label+n-m) extended load",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "table"},
			{Type: token.Plus},
			{Type: token.Number, Value: "3"},
			{Type: token.Minus},
			{Type: token.Number, Value: "1"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdExtended},
		wantVariant:    cpuz80.LdExtended,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegLoadExtA},
		wantValues: []ast.Node{
			expressionNode(
				token.Token{Type: token.Identifier, Value: "table"},
				token.Token{Type: token.Plus},
				token.Token{Type: token.Number, Value: "3"},
				token.Token{Type: token.Minus},
				token.Token{Type: token.Number, Value: "1"},
			),
		},
	},
	{
		name:     "ld a,(label+index) extended load",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "table"},
			{Type: token.Plus},
			{Type: token.Identifier, Value: "index"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdExtended},
		wantVariant:    cpuz80.LdExtended,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegLoadExtA},
		wantValues:     []ast.Node{expressionNode(token.Token{Type: token.Identifier, Value: "table"}, token.Token{Type: token.Plus}, token.Token{Type: token.Identifier, Value: "index"})},
	},
	{
		name:     "ld (nn),a extended store",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Number, Value: "$2345"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "a"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.LdExtended},
		wantVariant:    cpuz80.LdExtended,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegStoreExtA},
		wantValues:     []ast.Node{ast.NewNumber(0x2345)},
	},
	{
		name:     "ld bc,(nn) extended load register pair",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.Identifier, Value: "bc"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Number, Value: "$3456"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg16, cpuz80.EdLdBcNn},
		wantVariant:    cpuz80.EdLdBcNn,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegBC},
		wantValues:     []ast.Node{ast.NewNumber(0x3456)},
	},
	{
		name:     "ld (nn),bc extended store register pair",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Number, Value: "$4567"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "bc"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg16, cpuz80.EdLdNnBc},
		wantVariant:    cpuz80.EdLdNnBc,
		wantAddressing: cpuz80.ExtendedAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegBC},
		wantValues:     []ast.Node{ast.NewNumber(0x4567)},
	},
	{
		name:     "in a,(n) immediate port",
		mnemonic: cpuz80.InName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "in"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Number, Value: "$12"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.InPort, cpuz80.EdInAC},
		wantVariant:    cpuz80.InPort,
		wantAddressing: cpuz80.PortAddressing,
		wantValues:     []ast.Node{ast.NewNumber(0x12)},
	},
	{
		name:     "in a,(n+m) immediate port",
		mnemonic: cpuz80.InName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "in"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Number, Value: "$10"},
			{Type: token.Plus},
			{Type: token.Number, Value: "1"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.InPort, cpuz80.EdInAC},
		wantVariant:    cpuz80.InPort,
		wantAddressing: cpuz80.PortAddressing,
		wantValues:     []ast.Node{expressionNode(token.Token{Type: token.Number, Value: "$10"}, token.Token{Type: token.Plus}, token.Token{Type: token.Number, Value: "1"})},
	},
	{
		name:     "in a,(n+m-k) immediate port",
		mnemonic: cpuz80.InName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "in"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Number, Value: "$10"},
			{Type: token.Plus},
			{Type: token.Number, Value: "3"},
			{Type: token.Minus},
			{Type: token.Number, Value: "1"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.InPort, cpuz80.EdInAC},
		wantVariant:    cpuz80.InPort,
		wantAddressing: cpuz80.PortAddressing,
		wantValues: []ast.Node{
			expressionNode(
				token.Token{Type: token.Number, Value: "$10"},
				token.Token{Type: token.Plus},
				token.Token{Type: token.Number, Value: "3"},
				token.Token{Type: token.Minus},
				token.Token{Type: token.Number, Value: "1"},
			),
		},
	},
	{
		name:     "out (n),a immediate port",
		mnemonic: cpuz80.OutName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "out"},
			{Type: token.LeftParentheses},
			{Type: token.Number, Value: "$34"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "a"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.OutPort, cpuz80.EdOutCA},
		wantVariant:    cpuz80.OutPort,
		wantAddressing: cpuz80.PortAddressing,
		wantValues:     []ast.Node{ast.NewNumber(0x34)},
	},
	{
		name:     "in b,(c) port c register form",
		mnemonic: cpuz80.InName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "in"},
			{Type: token.Identifier, Value: "b"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "c"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.InPort, cpuz80.EdInBC},
		wantVariant:    cpuz80.EdInBC,
		wantAddressing: cpuz80.PortAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegB},
	},
	{
		name:     "out (c),e port c register form",
		mnemonic: cpuz80.OutName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "out"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "c"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "e"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.OutPort, cpuz80.EdOutCE},
		wantVariant:    cpuz80.EdOutCE,
		wantAddressing: cpuz80.PortAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegE},
	},
	{
		name:           "bit value first register",
		mnemonic:       cpuz80.BitName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "bit"}, {Type: token.Number, Value: "3"}, {Type: token.Comma}, {Type: token.Identifier, Value: "a"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.CBBit},
		wantVariant:    cpuz80.CBBit,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
		wantValues:     []ast.Node{ast.NewNumber(3)},
	},
	{
		name:     "ld a indexed ix displacement",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "ix"},
			{Type: token.Plus},
			{Type: token.Number, Value: "5"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.DdLdAIXd, cpuz80.FdLdAIYd},
		wantVariant:    cpuz80.DdLdAIXd,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
		wantValues:     []ast.Node{ast.NewNumber(5)},
	},
	{
		name:     "ld a indexed ix symbolic displacement",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "ix"},
			{Type: token.Plus},
			{Type: token.Identifier, Value: "disp"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.DdLdAIXd, cpuz80.FdLdAIYd},
		wantVariant:    cpuz80.DdLdAIXd,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
		wantValues:     []ast.Node{expressionNode(token.Token{Type: token.Identifier, Value: "disp"})},
	},
	{
		name:     "ld indexed iy displacement a",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "iy"},
			{Type: token.Minus},
			{Type: token.Number, Value: "2"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "a"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.DdLdIXdA, cpuz80.FdLdIYdA},
		wantVariant:    cpuz80.FdLdIYdA,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
		wantValues:     []ast.Node{ast.NewNumber(0xFE)},
	},
	{
		name:     "ld indexed iy compact minus form a",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "iy-2"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "a"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.DdLdIXdA, cpuz80.FdLdIYdA},
		wantVariant:    cpuz80.FdLdIYdA,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
		wantValues:     []ast.Node{ast.NewNumber(0xFE)},
	},
	{
		name:     "bit value first indexed ix",
		mnemonic: cpuz80.BitName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "bit"},
			{Type: token.Number, Value: "3"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "ix"},
			{Type: token.Plus},
			{Type: token.Number, Value: "5"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.CBBit, cpuz80.DdcbBit, cpuz80.FdcbBit},
		wantVariant:    cpuz80.DdcbBit,
		wantAddressing: cpuz80.BitAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegHLIndirect},
		wantValues:     []ast.Node{ast.NewNumber(3), ast.NewNumber(5)},
	},
	{
		name:     "bit value first indexed iy compact minus form",
		mnemonic: cpuz80.BitName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "bit"},
			{Type: token.Number, Value: "2"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "iy-1"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.CBBit, cpuz80.DdcbBit, cpuz80.FdcbBit},
		wantVariant:    cpuz80.FdcbBit,
		wantAddressing: cpuz80.BitAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegHLIndirect},
		wantValues:     []ast.Node{ast.NewNumber(2), ast.NewNumber(0xFF)},
	},
	{
		name:     "jp ix indirect",
		mnemonic: cpuz80.JpName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "jp"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "ix"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.JpAbs, cpuz80.JpIndirect, cpuz80.DdJpIX, cpuz80.FdJpIY},
		wantVariant:    cpuz80.DdJpIX,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegIX},
	},
	{
		name:     "inc indexed ix displacement",
		mnemonic: cpuz80.IncName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "inc"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "ix"},
			{Type: token.Plus},
			{Type: token.Number, Value: "1"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.IncIndirect, cpuz80.DdIncIXd, cpuz80.FdIncIYd},
		wantVariant:    cpuz80.DdIncIXd,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegIXIndirect},
		wantValues:     []ast.Node{ast.NewNumber(1)},
	},
	{
		name:           "im numeric register opcode variant",
		mnemonic:       cpuz80.ImName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "im"}, {Type: token.Number, Value: "1"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.EdIm0, cpuz80.EdIm1, cpuz80.EdIm2},
		wantVariant:    cpuz80.EdIm1,
		wantAddressing: cpuz80.ImmediateAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegIM1},
	},
	{
		name:           "rst numeric register opcode variant",
		mnemonic:       cpuz80.RstName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "rst"}, {Type: token.Number, Value: "$38"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.Rst},
		wantVariant:    cpuz80.Rst,
		wantAddressing: cpuz80.ImpliedAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegRst38},
	},

	// --- New resolver path tests ---

	// resolveNoOperand second pass: implied with RegisterOpcodes (NEG, RETN).
	{
		name:           "neg implied with register opcodes",
		mnemonic:       cpuz80.NegName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "neg"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.EdNeg},
		wantVariant:    cpuz80.EdNeg,
		wantAddressing: cpuz80.ImpliedAddressing,
	},
	{
		name:           "retn implied with register opcodes",
		mnemonic:       cpuz80.RetnName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "retn"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.EdRetn},
		wantVariant:    cpuz80.EdRetn,
		wantAddressing: cpuz80.ImpliedAddressing,
	},

	// resolveSingleOperand second pass: value addressing with RegisterOpcodes (SUB n).
	{
		name:           "sub immediate value",
		mnemonic:       cpuz80.SubName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "sub"}, {Type: token.Number, Value: "$01"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.SubA},
		wantVariant:    cpuz80.SubA,
		wantAddressing: cpuz80.ImmediateAddressing,
		wantValues:     []ast.Node{ast.NewNumber(0x01)},
	},

	// resolveSingleRegisterOperand indexed fallback: SUB (IX+d).
	{
		name:     "sub indexed ix displacement",
		mnemonic: cpuz80.SubName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "sub"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "ix"},
			{Type: token.Plus},
			{Type: token.Number, Value: "3"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.SubA, cpuz80.DdSubAIXd, cpuz80.FdSubAIYd},
		wantVariant:    cpuz80.DdSubAIXd,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
		wantValues:     []ast.Node{ast.NewNumber(3)},
	},

	// resolveAluRegisterPairOperands: ADD A,B (two-register ALU).
	{
		name:           "add a,b two register alu",
		mnemonic:       cpuz80.AddName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "add"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.Identifier, Value: "b"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.AddA, cpuz80.AddHl},
		wantVariant:    cpuz80.AddA,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegB},
	},

	// resolveAluRegisterPairOperands: ADD HL,BC.
	{
		name:           "add hl,bc register pair alu",
		mnemonic:       cpuz80.AddName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "add"}, {Type: token.Identifier, Value: "hl"}, {Type: token.Comma}, {Type: token.Identifier, Value: "bc"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.AddA, cpuz80.AddHl, cpuz80.DdAddIXBc},
		wantVariant:    cpuz80.AddHl,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegBC},
	},

	// resolveAluRegisterPairOperands: ADD IX,BC.
	{
		name:           "add ix,bc prefix alu",
		mnemonic:       cpuz80.AddName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "add"}, {Type: token.Identifier, Value: "ix"}, {Type: token.Comma}, {Type: token.Identifier, Value: "bc"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.AddA, cpuz80.AddHl, cpuz80.DdAddIXBc},
		wantVariant:    cpuz80.DdAddIXBc,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegBC},
	},

	// resolveAluRegisterPairOperands: SBC HL,BC.
	{
		name:           "sbc hl,bc register pair alu",
		mnemonic:       cpuz80.SbcName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "sbc"}, {Type: token.Identifier, Value: "hl"}, {Type: token.Comma}, {Type: token.Identifier, Value: "bc"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.SbcA, cpuz80.EdSbcHlBc},
		wantVariant:    cpuz80.EdSbcHlBc,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegBC},
	},

	// resolveIndirectLoadStoreOperands: LD A,(HL).
	{
		name:     "ld a,(hl) indirect load",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "hl"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8, cpuz80.LdIndirect},
		wantVariant:    cpuz80.LdReg8,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegLoadHLA},
	},

	// resolveIndirectLoadStoreOperands: LD (HL),A.
	{
		name:     "ld (hl),a indirect store",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "hl"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "a"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.LdIndirect, cpuz80.LdIndirectImm},
		wantVariant:    cpuz80.LdReg8,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegHLIndirect, cpuz80.RegA},
	},

	// resolveIndirectLoadStoreOperands: LD A,(BC).
	{
		name:     "ld a,(bc) indirect load",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "bc"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdIndirect},
		wantVariant:    cpuz80.LdIndirect,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegLoadBC},
	},

	// resolveIndirectImmediateOperands: LD (HL),n.
	{
		name:     "ld (hl),n indirect immediate",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "hl"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Number, Value: "$42"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.LdIndirect, cpuz80.LdIndirectImm},
		wantVariant:    cpuz80.LdIndirectImm,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantValues:     []ast.Node{ast.NewNumber(0x42)},
	},

	// resolveIndirectImmediateOperands: LD (IX+d),n.
	{
		name:     "ld (ix+d),n indexed immediate",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "ix"},
			{Type: token.Plus},
			{Type: token.Number, Value: "5"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Number, Value: "$42"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.DdLdIXdN, cpuz80.FdLdIYdN, cpuz80.DdLdIXdA, cpuz80.FdLdIYdA},
		wantVariant:    cpuz80.DdLdIXdN,
		wantAddressing: cpuz80.ImmediateAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegImm8},
		wantValues:     []ast.Node{ast.NewNumber(5), ast.NewNumber(0x42)},
	},

	// resolveIndirectImmediateOperands: LD (IY-d),n.
	{
		name:     "ld (iy-d),n indexed immediate",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "iy"},
			{Type: token.Minus},
			{Type: token.Number, Value: "8"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Number, Value: "$99"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.DdLdIXdN, cpuz80.FdLdIYdN, cpuz80.DdLdIXdA, cpuz80.FdLdIYdA},
		wantVariant:    cpuz80.FdLdIYdN,
		wantAddressing: cpuz80.ImmediateAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegIYIndirect},
		wantValues:     []ast.Node{ast.NewNumber(0xF8), ast.NewNumber(0x99)},
	},

	// resolveSpecialRegisterPairOperands: LD I,A.
	{
		name:           "ld i,a special register pair",
		mnemonic:       cpuz80.LdName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "i"}, {Type: token.Comma}, {Type: token.Identifier, Value: "a"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.EdLdIA},
		wantVariant:    cpuz80.EdLdIA,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegI},
	},

	// resolveSpecialRegisterPairOperands: LD A,R.
	{
		name:           "ld a,r special register pair",
		mnemonic:       cpuz80.LdName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.Identifier, Value: "r"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.EdLdAR},
		wantVariant:    cpuz80.EdLdAR,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
	},

	// resolveSpecialRegisterPairOperands: LD SP,HL.
	{
		name:           "ld sp,hl special register pair",
		mnemonic:       cpuz80.LdName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "sp"}, {Type: token.Comma}, {Type: token.Identifier, Value: "hl"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg16, cpuz80.LdSp, cpuz80.DdLdSpIX, cpuz80.FdLdSpIY},
		wantVariant:    cpuz80.LdSp,
		wantAddressing: cpuz80.RegisterAddressing,
	},

	// resolveSpecialRegisterPairOperands: LD SP,IX.
	{
		name:           "ld sp,ix special register pair",
		mnemonic:       cpuz80.LdName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "sp"}, {Type: token.Comma}, {Type: token.Identifier, Value: "ix"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg16, cpuz80.LdSp, cpuz80.DdLdSpIX, cpuz80.FdLdSpIY},
		wantVariant:    cpuz80.DdLdSpIX,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegIX},
	},

	// resolveSpecialRegisterPairOperands: LD SP,IY.
	{
		name:           "ld sp,iy special register pair",
		mnemonic:       cpuz80.LdName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "sp"}, {Type: token.Comma}, {Type: token.Identifier, Value: "iy"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg16, cpuz80.LdSp, cpuz80.DdLdSpIX, cpuz80.FdLdSpIY},
		wantVariant:    cpuz80.FdLdSpIY,
		wantAddressing: cpuz80.RegisterAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegIY},
	},

	// resolveSpecialRegisterPairOperands: EX DE,HL.
	{
		name:           "ex de,hl special register pair",
		mnemonic:       cpuz80.ExName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ex"}, {Type: token.Identifier, Value: "de"}, {Type: token.Comma}, {Type: token.Identifier, Value: "hl"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.ExDeHl, cpuz80.ExAf, cpuz80.ExSp, cpuz80.DdExSpIX, cpuz80.FdExSpIY},
		wantVariant:    cpuz80.ExDeHl,
		wantAddressing: cpuz80.ImpliedAddressing,
	},

	// resolveSpecialRegisterPairOperands: EX AF,AF.
	{
		name:           "ex af,af special register pair",
		mnemonic:       cpuz80.ExName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ex"}, {Type: token.Identifier, Value: "af"}, {Type: token.Comma}, {Type: token.Identifier, Value: "af"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.ExDeHl, cpuz80.ExAf, cpuz80.ExSp},
		wantVariant:    cpuz80.ExAf,
		wantAddressing: cpuz80.ImpliedAddressing,
	},

	// resolveIndirectLoadStoreOperands fallback: EX (SP),HL.
	{
		name:     "ex (sp),hl indirect load store fallback",
		mnemonic: cpuz80.ExName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ex"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "sp"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "hl"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.ExDeHl, cpuz80.ExAf, cpuz80.ExSp, cpuz80.DdExSpIX, cpuz80.FdExSpIY},
		wantVariant:    cpuz80.ExSp,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
	},

	// resolveRegisterValueOperands: ADD A,42 (ALU RegA stripping).
	{
		name:           "add a,n alu immediate strips rega",
		mnemonic:       cpuz80.AddName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "add"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.Number, Value: "42"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.AddA, cpuz80.AddHl},
		wantVariant:    cpuz80.AddA,
		wantAddressing: cpuz80.ImmediateAddressing,
		wantValues:     []ast.Node{ast.NewNumber(42)},
	},

	// resolveRegisterValueOperands: LD A,42 preserves RegisterParams.
	{
		name:           "ld a,n preserves register param",
		mnemonic:       cpuz80.LdName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.Number, Value: "42"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8},
		wantVariant:    cpuz80.LdImm8,
		wantAddressing: cpuz80.ImmediateAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
		wantValues:     []ast.Node{ast.NewNumber(42)},
	},

	// resolveRegisterValueOperands: ADC A,$FF immediate.
	{
		name:           "adc a,n alu immediate",
		mnemonic:       cpuz80.AdcName,
		tokens:         []token.Token{{Type: token.Identifier, Value: "adc"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.Number, Value: "$FF"}, {Type: token.EOL}},
		variants:       []*cpuz80.Instruction{cpuz80.AdcA, cpuz80.EdAdcHlBc},
		wantVariant:    cpuz80.AdcA,
		wantAddressing: cpuz80.ImmediateAddressing,
		wantValues:     []ast.Node{ast.NewNumber(0xFF)},
	},

	// resolveIndirectImmediateOperands does NOT match LD (IY-2),A (register, not value).
	{
		name:     "ld (iy-d),a indexed store not confused with immediate",
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "iy"},
			{Type: token.Minus},
			{Type: token.Number, Value: "2"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "a"},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.LdReg8, cpuz80.DdLdIXdA, cpuz80.FdLdIYdA, cpuz80.DdLdIXdN, cpuz80.FdLdIYdN},
		wantVariant:    cpuz80.FdLdIYdA,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
		wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
		wantValues:     []ast.Node{ast.NewNumber(0xFE)},
	},

	// JP (HL) parenthesized indirect fallback.
	{
		name:     "jp (hl) indirect fallback",
		mnemonic: cpuz80.JpName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "jp"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "hl"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants:       []*cpuz80.Instruction{cpuz80.JpAbs, cpuz80.JpCond, cpuz80.JpIndirect, cpuz80.DdJpIX, cpuz80.FdJpIY},
		wantVariant:    cpuz80.JpIndirect,
		wantAddressing: cpuz80.RegisterIndirectAddressing,
	},
}

func TestParseIdentifier_MinimumSlice(t *testing.T) {
	for _, tt := range parseIdentifierTests {
		t.Run(tt.name, func(t *testing.T) {
			parser := newMockParser(tt.tokens...)
			node, err := ParseIdentifier(parser, tt.mnemonic, tt.variants)
			assert.NoError(t, err)

			ins, ok := node.(ast.Instruction)
			assert.True(t, ok)
			assert.Equal(t, tt.mnemonic, ins.Name)
			assert.Equal(t, int(tt.wantAddressing), ins.Addressing)

			typedArg, ok := ins.Argument.(ast.InstructionArgument)
			assert.True(t, ok)

			resolved, ok := typedArg.Value.(ResolvedInstruction)
			assert.True(t, ok)
			assert.Equal(t, tt.wantVariant, resolved.Instruction)
			assert.Equal(t, tt.wantAddressing, resolved.Addressing)
			assert.Equal(t, tt.wantRegister, resolved.RegisterParams)
			if len(tt.wantValues) == 0 {
				assert.Empty(t, resolved.OperandValues)
				return
			}
			assert.Equal(t, tt.wantValues, resolved.OperandValues)
		})
	}
}

func TestParseIdentifier_Errors(t *testing.T) {
	for _, tt := range parseIdentifierErrorCases() {
		t.Run(tt.name, func(t *testing.T) {
			parser := newMockParser(tt.tokens...)
			_, err := ParseIdentifier(parser, tt.mnemonic, tt.variants)
			assert.Error(t, err)
		})
	}
}

func TestParseIdentifier_DiagnosticMessageConditionCAmbiguity(t *testing.T) {
	assertDiagnosticError(t, diagnosticCase{
		mnemonic: cpuz80.JpName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "jp"},
			{Type: token.Identifier, Value: "c"},
			{Type: token.Comma},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "hl"},
			{Type: token.RightParentheses},
			{Type: token.EOL},
		},
		variants: []*cpuz80.Instruction{
			cpuz80.JpCond,
			cpuz80.JpAbs,
			cpuz80.JpIndirect,
			cpuz80.DdJpIX,
			cpuz80.FdJpIY,
		},
		errorContain: []string{"ambiguous operand 'c'", "carry condition or register c"},
	})
}

func TestParseIdentifier_DiagnosticMessageImmediateVsAddressed(t *testing.T) {
	assertDiagnosticError(t, diagnosticCase{
		mnemonic: cpuz80.InName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "in"},
			{Type: token.Identifier, Value: "a"},
			{Type: token.Comma},
			{Type: token.Number, Value: "$12"},
			{Type: token.EOL},
		},
		variants: []*cpuz80.Instruction{
			cpuz80.InPort,
			cpuz80.EdInAC,
		},
		errorContain: []string{"immediate vs addressed operand mismatch", "parenthesized forms"},
	})
}

func TestParseIdentifier_DiagnosticMessageIndexedLoadDirection(t *testing.T) {
	assertDiagnosticError(t, diagnosticCase{
		mnemonic: cpuz80.LdName,
		tokens: []token.Token{
			{Type: token.Identifier, Value: "ld"},
			{Type: token.LeftParentheses},
			{Type: token.Identifier, Value: "ix"},
			{Type: token.Plus},
			{Type: token.Number, Value: "1"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "ix"},
			{Type: token.EOL},
		},
		variants: []*cpuz80.Instruction{
			cpuz80.LdReg8,
			cpuz80.DdLdIXdA,
			cpuz80.FdLdIYdA,
			cpuz80.DdLdAIXd,
			cpuz80.FdLdAIYd,
		},
		errorContain: []string{"indexed load direction mismatch", "ld r,(ix+d|iy+d)"},
	})
}

type diagnosticCase struct {
	mnemonic     string
	tokens       []token.Token
	variants     []*cpuz80.Instruction
	errorContain []string
}

type parseIdentifierErrorCase struct {
	name     string
	mnemonic string
	tokens   []token.Token
	variants []*cpuz80.Instruction
}

func assertDiagnosticError(t *testing.T, tc diagnosticCase) {
	t.Helper()

	parser := newMockParser(tc.tokens...)
	_, err := ParseIdentifier(parser, tc.mnemonic, tc.variants)
	assert.Error(t, err)

	for _, fragment := range tc.errorContain {
		assert.ErrorContains(t, err, fragment)
	}
}

func parseIdentifierErrorCases() []parseIdentifierErrorCase {
	cases := parseIdentifierErrorCasesBase()
	cases = append(cases, parseIdentifierErrorCasesPortAndOffset()...)
	return cases
}

func parseIdentifierErrorCasesBase() []parseIdentifierErrorCase {
	return []parseIdentifierErrorCase{
		{
			name:     "missing second operand after comma",
			mnemonic: cpuz80.LdName,
			tokens:   []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.EOL}},
			variants: []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8},
		},
		{
			name:     "unsupported operand pattern",
			mnemonic: cpuz80.JpName,
			tokens:   []token.Token{{Type: token.Identifier, Value: "jp"}, {Type: token.Number, Value: "1"}, {Type: token.Comma}, {Type: token.Number, Value: "2"}, {Type: token.EOL}},
			variants: []*cpuz80.Instruction{cpuz80.JpAbs, cpuz80.JpCond},
		},
		{
			name:     "unsupported indexed base register",
			mnemonic: cpuz80.BitName,
			tokens: []token.Token{
				{Type: token.Identifier, Value: "bit"},
				{Type: token.Number, Value: "3"},
				{Type: token.Comma},
				{Type: token.LeftParentheses},
				{Type: token.Identifier, Value: "hl"},
				{Type: token.Plus},
				{Type: token.Number, Value: "1"},
				{Type: token.RightParentheses},
				{Type: token.EOL},
			},
			variants: []*cpuz80.Instruction{cpuz80.CBBit, cpuz80.DdcbBit},
		},
		{
			name:     "missing closing parenthesis",
			mnemonic: cpuz80.JpName,
			tokens: []token.Token{
				{Type: token.Identifier, Value: "jp"},
				{Type: token.LeftParentheses},
				{Type: token.Identifier, Value: "ix"},
				{Type: token.EOL},
			},
			variants: []*cpuz80.Instruction{cpuz80.DdJpIX},
		},
	}
}

func parseIdentifierErrorCasesPortAndOffset() []parseIdentifierErrorCase {
	return []parseIdentifierErrorCase{
		{
			name:     "out immediate with non-a register is invalid",
			mnemonic: cpuz80.OutName,
			tokens: []token.Token{
				{Type: token.Identifier, Value: "out"},
				{Type: token.LeftParentheses},
				{Type: token.Number, Value: "$34"},
				{Type: token.RightParentheses},
				{Type: token.Comma},
				{Type: token.Identifier, Value: "b"},
				{Type: token.EOL},
			},
			variants: []*cpuz80.Instruction{cpuz80.OutPort, cpuz80.EdOutCB},
		},
		{
			name:     "offset expression missing operator between values",
			mnemonic: cpuz80.JpName,
			tokens: []token.Token{
				{Type: token.Identifier, Value: "jp"},
				{Type: token.Identifier, Value: "target"},
				{Type: token.Plus},
				{Type: token.Identifier, Value: "next"},
				{Type: token.Identifier, Value: "extra"},
				{Type: token.EOL},
			},
			variants: []*cpuz80.Instruction{cpuz80.JpAbs},
		},
		{
			name:     "offset operator missing value",
			mnemonic: cpuz80.JpName,
			tokens: []token.Token{
				{Type: token.Identifier, Value: "jp"},
				{Type: token.Identifier, Value: "target"},
				{Type: token.Plus},
				{Type: token.EOL},
			},
			variants: []*cpuz80.Instruction{cpuz80.JpAbs},
		},
	}
}

func expressionNode(tokens ...token.Token) ast.Node {
	return ast.NewExpression(tokens...)
}
