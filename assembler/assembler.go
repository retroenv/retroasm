// Package assembler implements the assembler functionality
package assembler

import (
	"fmt"
	"io"
	"os"

	"github.com/retroenv/retroasm/assembler/config"
	"github.com/retroenv/retroasm/scope"
)

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
	steps := asm.Steps()
	for i, stp := range steps {
		if err := stp.handler(asm); err != nil {
			return fmt.Errorf("executing assembler step %d/%d: %s: %w",
				i+1, len(steps), stp.errorTemplate, err)
		}
	}
	return nil
}
