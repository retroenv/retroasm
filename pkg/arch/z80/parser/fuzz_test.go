package parser

import (
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func FuzzParseIdentifier_NoPanic(f *testing.F) {
	seeds := [][]byte{
		{0x00, 0x10, 0x40, 0x80},
		{0x01, 0x20, 0x30, 0x40, 0x50},
		{0x02, 0x05, 0x08, 0x11, 0x09},
		{0x03, 0x10, 0x04, 0x30, 0x01, 0x02},
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			t.Skip()
		}

		mnemonic, variants := fuzzMnemonicAndVariants(data[0])
		tokens := fuzzTokens(mnemonic, data[1:])

		parser1 := newMockParser(tokens...)
		node1, err1 := ParseIdentifier(parser1, mnemonic, variants)

		// Property: parsing is deterministic for the same token stream.
		parser2 := newMockParser(tokens...)
		node2, err2 := ParseIdentifier(parser2, mnemonic, variants)

		assert.Equal(t, err1 == nil, err2 == nil)

		if err1 != nil && err2 != nil {
			assert.Equal(t, err1.Error(), err2.Error())
			return
		}

		assert.Equal(t, node1 == nil, node2 == nil)
	})
}

func fuzzMnemonicAndVariants(selector byte) (string, []*cpuz80.Instruction) {
	switch selector % 5 {
	case 0:
		return cpuz80.LdName, []*cpuz80.Instruction{
			cpuz80.LdImm8,
			cpuz80.LdReg8,
			cpuz80.LdReg16,
			cpuz80.LdExtended,
			cpuz80.DdLdAIXd,
			cpuz80.DdLdIXdA,
			cpuz80.FdLdAIYd,
			cpuz80.FdLdIYdA,
		}
	case 1:
		return cpuz80.JpName, []*cpuz80.Instruction{
			cpuz80.JpAbs,
			cpuz80.JpCond,
			cpuz80.JpIndirect,
			cpuz80.DdJpIX,
			cpuz80.FdJpIY,
		}
	case 2:
		return cpuz80.BitName, []*cpuz80.Instruction{
			cpuz80.CBBit,
			cpuz80.DdcbBit,
			cpuz80.FdcbBit,
		}
	case 3:
		return cpuz80.InName, []*cpuz80.Instruction{
			cpuz80.InPort,
			cpuz80.EdInAC,
			cpuz80.EdInBC,
		}
	default:
		return cpuz80.OutName, []*cpuz80.Instruction{
			cpuz80.OutPort,
			cpuz80.EdOutCA,
			cpuz80.EdOutCE,
		}
	}
}

func fuzzTokens(mnemonic string, payload []byte) []token.Token {
	tokens := []token.Token{{Type: token.Identifier, Value: strings.ToLower(mnemonic)}}
	if len(payload) == 0 {
		return append(tokens, token.Token{Type: token.EOL})
	}

	for _, raw := range payload {
		tokens = append(tokens, fuzzToken(raw))
		if len(tokens) >= 10 {
			break
		}
	}

	return append(tokens, token.Token{Type: token.EOL})
}

func fuzzToken(raw byte) token.Token {
	identifiers := []string{"a", "b", "c", "hl", "ix", "iy", "nz", "z", "target", "table", "disp"}
	numbers := []string{"0", "1", "2", "3", "5", "$10", "$80", "$FF", "$1234"}

	switch raw % 7 {
	case 0:
		return token.Token{Type: token.Identifier, Value: identifiers[int(raw)%len(identifiers)]}
	case 1:
		return token.Token{Type: token.Number, Value: numbers[int(raw)%len(numbers)]}
	case 2:
		return token.Token{Type: token.LeftParentheses}
	case 3:
		return token.Token{Type: token.RightParentheses}
	case 4:
		return token.Token{Type: token.Comma}
	case 5:
		return token.Token{Type: token.Plus}
	default:
		return token.Token{Type: token.Minus}
	}
}
