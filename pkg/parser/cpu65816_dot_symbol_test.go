package parser_test

import (
	"strings"
	"testing"

	asmcpu65816 "github.com/retroenv/retroasm/pkg/arch/cpu65816"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	asmparser "github.com/retroenv/retroasm/pkg/parser"
	"github.com/retroenv/retrogolib/assert"
)

func TestCPU65816DotSymbolRejectsMissingIdentifier(t *testing.T) {
	for _, source := range []string{"lda f:.,x", "lda f:.1,x", "jmp ."} {
		t.Run(source, func(t *testing.T) {
			parser := asmparser.New(asmcpu65816.New().Arch, strings.NewReader(source), config.CompatDefault)
			assert.NoError(t, parser.Read(t.Context()))
			_, err := parser.TokensToAstNodes()
			assert.Error(t, err)
		})
	}
}
