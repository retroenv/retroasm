// Package assembler implements the assembler functionality
package assembler

import (
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

// Assembler is the assembler implementation.
type Assembler[T any] struct {
	cfg         *config.Config[T]
	inputReader io.Reader
	writer      io.Writer

	// a function that reads in a file, for testing includes, defaults to os.ReadFile
	fileReader func(name string) ([]byte, error)

	fileScope *scope.Scope // scope for current to be parsed file

	segments      map[string]*segment // maps segment name to segment
	segmentsOrder []*segment          // sorted list of all parsed segments

	macros map[string]macro
}

// New returns a new assembler.
func New[T any](cfg *config.Config[T], reader io.Reader, writer io.Writer) *Assembler[T] {
	return &Assembler[T]{
		cfg:         cfg,
		inputReader: reader,
		writer:      writer,

		fileReader: os.ReadFile,

		fileScope: scope.New(nil),

		macros: map[string]macro{},
	}
}

// Process the input file and assemble it into the writer output.
func (asm *Assembler[T]) Process() error {
	// Parse AST nodes first
	pars := parser.New[T](asm.cfg.Arch, asm.inputReader)
	if err := pars.Read(); err != nil {
		return fmt.Errorf("parsing lexer tokens: %w", err)
	}
	nodes, err := pars.TokensToAstNodes()
	if err != nil {
		return fmt.Errorf("converting tokens to ast nodes: %w", err)
	}

	return asm.ProcessAST(nodes)
}

// ProcessAST processes the given AST nodes and assembles them into the writer output.
func (asm *Assembler[T]) ProcessAST(nodes []ast.Node) error {
	// First process the AST nodes
	if err := asm.parseASTNodes(nodes); err != nil {
		return fmt.Errorf("parsing AST nodes: %w", err)
	}

	// Then run the remaining assembly steps
	steps := asm.Steps()
	for i, stp := range steps {
		if err := stp.handler(asm); err != nil {
			return fmt.Errorf("executing assembler step %d/%d: %s: %w",
				i+1, len(steps), stp.errorTemplate, err)
		}
	}
	return nil
}

// parseASTNodes processes the given AST nodes and converts them to internal types.
func (asm *Assembler[T]) parseASTNodes(nodes []ast.Node) error {
	p := &parseAST[T]{
		cfg:          asm.cfg,
		fileReader:   asm.fileReader,
		currentScope: asm.fileScope,
		segments:     map[string]*segment{},
	}

	for _, node := range nodes {
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
