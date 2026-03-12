// Package directives contains assembler directives parser.
//
// This package implements parsing for assembly directives (commands starting with '.')
// that control the assembler behavior. Supported directive categories include:
//   - Data: .byte, .word, .db, .dw (data definition)
//   - Storage: .dsb, .dsw, .res (reserved space)
//   - Organization: .org, .base, .align, .pad (memory layout)
//   - Conditionals: .if/.else/.endif, .ifdef/.ifndef (conditional assembly)
//   - Macros: .macro/.endm, .rept/.endr (code generation)
//   - Includes: .include, .incbin (file inclusion)
//   - Configuration: .segment, .bank, .setcpu (assembler settings)
//
// The BuildHandlers function provides mode-specific directive dispatch maps.
// Each handler receives a parser instance and returns the corresponding AST node.
package directives

import (
	"errors"
	"maps"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/set"
)

var (
	errMissingParameter    = errors.New("missing parameter")
	errUnexpectedParameter = errors.New("unexpected parameter")
)

// Handler defines a handler for an assembler directive.
type Handler func(arch.Parser) (ast.Node, error)

// Handlers maps the assembler directives to their handler function.
//
// Deprecated: Use BuildHandlers for mode-specific directive maps.
var Handlers = baseHandlers()

// BuildHandlers returns a directive handler map for the given compatibility mode.
// The base map contains universally supported directives; each mode overlays its specific additions.
func BuildHandlers(mode config.CompatibilityMode) map[string]Handler {
	handlers := baseHandlers()

	switch mode {
	case config.CompatX816:
		mergeHandlers(handlers, x816Handlers())
	case config.CompatAsm6:
		mergeHandlers(handlers, asm6Handlers())
	case config.CompatCa65:
		mergeHandlers(handlers, ca65Handlers())
	case config.CompatNesasm:
		mergeHandlers(handlers, nesasmHandlers())
	}

	return handlers
}

func baseHandlers() map[string]Handler {
	return map[string]Handler{
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
}

func x816Handlers() map[string]Handler {
	return map[string]Handler{
		// x816 uses .dcb for string/data definition (already in base)
		// No-op directives for listing/display
		"list":    NoOp,
		"nolist":  NoOp,
		"sym":     NoOp,
		"cerror":  NoOp,
		"cwarn":   NoOp,
		"message": NoOp,
	}
}

func asm6Handlers() map[string]Handler {
	// asm6 directives are mostly in base already
	return map[string]Handler{}
}

func ca65Handlers() map[string]Handler {
	return map[string]Handler{
		// No-op directives for ca65
		"list":       NoOp,
		"listbytes":  NoOp,
		"debuginfo":  NoOp,
		"export":     NoOp,
		"exportzp":   NoOp,
		"import":     NoOp,
		"importzp":   NoOp,
		"feature":    NoOp,
		"charmap":    NoOp,
		"autoimport": NoOp,
	}
}

func nesasmHandlers() map[string]Handler {
	return map[string]Handler{
		// No-op directives for NESASM
		"list": NoOp,
	}
}

func mergeHandlers(dst, src map[string]Handler) {
	maps.Copy(dst, src)
}

var directiveBinaryIncludes = set.NewFromSlice([]string{
	"bin",    // asm6
	"incbin", // asm6
})

// SetCPU skips the .setcpu directive as it is not currently used.
//
//nolint:nilnil // directive is intentionally ignored
func SetCPU(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(2)
	return nil, nil
}

// NoOp is a directive handler that consumes all tokens until end of line.
// Used for directives that don't affect binary output (listing, display, symbol file output).
//
//nolint:nilnil // directive is intentionally ignored
func NoOp(p arch.Parser) (ast.Node, error) {
	for {
		p.AdvanceReadPosition(1)
		if p.NextToken(0).Type.IsTerminator() {
			return nil, nil
		}
	}
}
