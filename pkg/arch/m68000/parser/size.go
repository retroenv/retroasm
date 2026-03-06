package parser

import (
	"strings"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

// ParseSizeSuffix extracts the size suffix from a mnemonic.
// e.g., "MOVE.L" returns ("MOVE", SizeLong), "ADD.B" returns ("ADD", SizeByte).
func ParseSizeSuffix(mnemonic string) (string, m68000.OperandSize) {
	idx := strings.LastIndex(mnemonic, ".")
	if idx < 0 || idx+2 != len(mnemonic) {
		return mnemonic, 0
	}

	suffix := strings.ToUpper(mnemonic[idx+1:])
	base := mnemonic[:idx]

	switch suffix {
	case "B":
		return base, m68000.SizeByte
	case "W":
		return base, m68000.SizeWord
	case "L":
		return base, m68000.SizeLong
	default:
		return mnemonic, 0
	}
}

// parseSizeToken returns the operand size from a token value (B/W/L).
func parseSizeToken(tok token.Token) m68000.OperandSize {
	if tok.Type != token.Identifier {
		return 0
	}
	switch strings.ToUpper(tok.Value) {
	case "B":
		return m68000.SizeByte
	case "W":
		return m68000.SizeWord
	case "L":
		return m68000.SizeLong
	default:
		return 0
	}
}
