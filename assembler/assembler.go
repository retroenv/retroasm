// Package assembler implements the assembler functionality for retrocomputers.
//
// The assembler converts assembly language source code into machine code for
// retro computer systems. It supports multiple assembly formats (asm6, ca65, nesasm)
// and follows a multi-stage pipeline architecture:
//
// 1. Parse AST nodes (text → AST or direct AST input)
// 2. Process macros and expand definitions
// 3. Evaluate expressions and resolve symbols
// 4. Update data sizes based on expressions
// 5. Assign memory addresses to instructions and data
// 6. Generate opcodes for target architecture
// 7. Write final output to memory segments
//
// The assembler is designed for both library integration (AST-first) and
// CLI usage (text-based), providing flexible APIs for different use cases.
package assembler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/retroenv/retroasm/assembler/config"
	"github.com/retroenv/retroasm/parser"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retroasm/scope"
)

var errNoCurrentSegment = errors.New("no current segment found")

// Assembler is the assembler implementation for retro computer systems.
// It processes assembly language and converts it into machine code through
// a multi-stage pipeline. The generic type T represents the target architecture.
type Assembler[T any] struct {
	cfg    *config.Config[T]
	writer io.Writer

	// a function that reads in a file, for testing includes, defaults to os.ReadFile
	fileReader func(name string) ([]byte, error)

	fileScope *scope.Scope // scope for current to be parsed file

	segments      map[string]*segment // maps segment name to segment
	segmentsOrder []*segment          // sorted list of all parsed segments

	macros map[string]macro
}

// New returns a new assembler.
func New[T any](cfg *config.Config[T], writer io.Writer) *Assembler[T] {
	return &Assembler[T]{
		cfg:    cfg,
		writer: writer,

		fileReader: os.ReadFile,

		fileScope: scope.New(nil),

		macros: map[string]macro{},
	}
}

// Process processes assembly source code from a reader and assembles it into the output writer.
// This is the primary text-based API for CLI usage. For library integration with
// pre-parsed AST nodes, use ProcessAST instead.
func (asm *Assembler[T]) Process(ctx context.Context, inputReader io.Reader) error {
	// Parse AST nodes first
	pars := parser.New[T](asm.cfg.Arch, inputReader)
	if err := pars.Read(ctx); err != nil {
		return fmt.Errorf("parsing lexer tokens: %w", err)
	}
	nodes, err := pars.TokensToAstNodes()
	if err != nil {
		return fmt.Errorf("converting tokens to ast nodes: %w", err)
	}

	return asm.ProcessAST(ctx, nodes)
}

// ProcessAST processes pre-parsed AST nodes and assembles them into the output writer.
// This is the primary AST-based API for library integration where AST nodes are
// already available. For text-based assembly from readers, use Process instead.
func (asm *Assembler[T]) ProcessAST(ctx context.Context, nodes []ast.Node) error {
	// First process the AST nodes
	if err := asm.parseASTNodes(ctx, nodes); err != nil {
		return fmt.Errorf("parsing AST nodes: %w", err)
	}

	// Then run the remaining assembly steps
	steps := asm.Steps()
	for i, stp := range steps {
		// Check for cancellation before each step
		select {
		case <-ctx.Done():
			return fmt.Errorf("processing cancelled: %w", ctx.Err())
		default:
		}
		if err := stp.handler(asm); err != nil {
			return fmt.Errorf("executing assembler step %d/%d: %s: %w",
				i+1, len(steps), stp.errorTemplate, err)
		}
	}
	return nil
}

// parseASTNodes processes the given AST nodes and converts them to internal types.
func (asm *Assembler[T]) parseASTNodes(ctx context.Context, nodes []ast.Node) error {
	p := &parseAST[T]{
		cfg:          asm.cfg,
		fileReader:   asm.fileReader,
		currentScope: asm.fileScope,
		segments:     map[string]*segment{},
	}

	for _, node := range nodes {
		// Check for cancellation in the parsing loop
		select {
		case <-ctx.Done():
			return fmt.Errorf("AST parsing cancelled: %w", ctx.Err())
		default:
		}
		switch n := node.(type) {
		case *ast.Comment:

		case ast.Segment:
			if err := parseSegment(p, n); err != nil {
				return fmt.Errorf("parsing segment node: %w", err)
			}

		default:
			if p.currentSegment == nil {
				return errNoCurrentSegment
			}

			newNodes, err := parseASTNode(p, node)
			if err != nil {
				return err
			}
			for _, newNode := range newNodes {
				p.currentSegment.addNode(newNode)
			}
		}
	}

	asm.segments = p.segments
	asm.segmentsOrder = p.segmentsOrder

	return nil
}
