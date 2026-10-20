package parser_test

import (
	"strings"
	"testing"

	asmsm83 "github.com/retroenv/retroasm/pkg/arch/sm83"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	asmparser "github.com/retroenv/retroasm/pkg/parser"
	"github.com/retroenv/retrogolib/assert"
)

func TestSM83DotSymbolRejectsMissingIdentifier(t *testing.T) {
	for _, source := range []string{"ld hl,.", "ld a,(.)", "jp .1"} {
		t.Run(source, func(t *testing.T) {
			parser := asmparser.New(asmsm83.New().Arch, strings.NewReader(source), config.CompatDefault)
			assert.NoError(t, parser.Read(t.Context()))
			_, err := parser.TokensToAstNodes()
			assert.Error(t, err)
		})
	}
}
