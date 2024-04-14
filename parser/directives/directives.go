// Package directives contains assembler directives parser.
package directives

import (
	"errors"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/parser/ast"
)

// Parser defines the parser that the directives use to read tokens.
type Parser interface {
	AdvanceReadPosition(offset int)
	Arch() arch.Architecture
	NextToken(offset int) token.Token
}

var (
	errMissingParameter    = errors.New("missing parameter")
	errUnexpectedParameter = errors.New("unexpected parameter")
)

// Handler defines a handler for an assembler directive.
type Handler func(Parser) (ast.Node, error)

// Handlers maps the assembler directives to their handler function.
var Handlers = map[string]Handler{
	"addr":       Addr,
	"align":      Align, // asm6
	"bank":       Bank,
	"base":       Base,
	"bin":        Include, // asm6
	"byt":        Data,
	"byte":       Data,     // asm6
	"db":         Data,     // asm6
	"dcb":        Data,     // asm6
	"dcw":        Data,     // asm6
	"dh":         AddrHigh, // asm6
	"dl":         AddrLow,  // asm6
	"dsb":        DataStorage,
	"dsw":        DataStorage,
	"dw":         Data,   // asm6
	"else":       Else,   // asm6
	"elseif":     Elseif, // asm6
	"endif":      Endif,  // asm6
	"ende":       Ende,   // asm6
	"endproc":    EndProc,
	"endr":       Endr,      // asm6
	"enum":       Enum,      // asm6
	"error":      Error,     // asm6
	"fillvalue":  FillValue, // asm6
	"hex":        Hex,       // asm6
	"if":         If,        // asm6
	"ifdef":      Ifdef,     // asm6
	"ifndef":     Ifndef,    // asm6
	"incbin":     Include,   // asm6
	"include":    Include,   // asm6
	"incsrc":     Include,   // asm6
	"inesbat":    NesasmConfig,
	"ineschr":    NesasmConfig,
	"inesmap":    NesasmConfig,
	"inesmir":    NesasmConfig,
	"inesprg":    NesasmConfig,
	"inessubmap": NesasmConfig,
	"macro":      Macro,   // asm6
	"org":        Base,    // asm6
	"pad":        Padding, // asm6
	"proc":       Proc,
	"rept":       Rept, // asm6
	"res":        Res,
	"rsset":      NesasmOffsetCounter,
	"segment":    Segment,
	"setcpu":     SetCPU,
	"word":       Data, // asm6
}

var directiveBinaryIncludes = map[string]struct{}{
	"bin":    {}, // asm6
	"incbin": {}, // asm6
}

// SetCPU ...
func SetCPU(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(2)
	return nil, nil
}
