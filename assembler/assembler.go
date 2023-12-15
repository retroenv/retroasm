// Package assembler implements the assembler functionality
package assembler

import (
	"fmt"
	"io"
	"os"

	"github.com/retroenv/assembler/assembler/config"
	"github.com/retroenv/assembler/parser"
	"github.com/retroenv/assembler/scope"
)

// Assembler is the assembler implementation.
type Assembler struct {
	cfg    *config.Config
	parser *parser.Parser
	writer io.Writer

	fileReader func(name string) ([]byte, error)

	fileScope    *scope.Scope // scope for current to be parsed file
	currentScope *scope.Scope // current scope, can be a function scope with file scope as parent

	currentSegment *segment            // the current segment being parsed
	segments       map[string]*segment // maps segment name to segment
	segmentsOrder  []*segment          // sorted list of all parsed segments

	currentContext *context
}

// New returns a new assembler.
func New(cfg *config.Config, reader io.Reader, writer io.Writer) *Assembler {
	sc := scope.New(nil)

	return &Assembler{
		cfg:    cfg,
		parser: parser.New(cfg.Arch, reader),
		writer: writer,

		fileReader: os.ReadFile,

		fileScope:      sc,
		currentScope:   sc,
		currentSegment: nil,
		segments:       map[string]*segment{},
		segmentsOrder:  nil,

		currentContext: &context{
			processNodes: true,
			parent:       nil,
		},
	}
}

// Process the input file and assemble it into the writer output.
func (asm *Assembler) Process() error {
	for i, stp := range steps {
		if err := stp.handler(asm); err != nil {
			return fmt.Errorf("executing assembler step %d/%d: %s: %w",
				i+1, len(steps), stp.errorTemplate, err)
		}
	}
	return nil
}
