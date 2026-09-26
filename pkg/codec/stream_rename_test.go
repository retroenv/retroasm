package codec_test

import (
	"strings"
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestCodecRenameSymbolsAcrossArchitectures(t *testing.T) {
	for _, test := range streamEquivalenceCases() {
		t.Run(test.name, func(t *testing.T) {
			c := test.newCodec(t)
			parsed, err := c.ParseStream(t.Context(), equivalenceInputSource, strings.NewReader(test.source))
			assert.NoError(t, err)
			baseline, err := c.AssembleStream(t.Context(), parsed)
			assert.NoError(t, err)
			for _, original := range []streamVariant{{name: "parsed", stream: parsed}, {name: "assembled", stream: baseline.Stream}} {
				t.Run(original.name, func(t *testing.T) {
					before := original.stream.Copy()
					renamed := original.stream.Copy()
					assert.NoError(t, renamed.RenameSymbols(map[string]string{"entry": "renamed_entry", "target": "renamed_target"}))
					assert.NoError(t, c.ValidateStream(renamed))
					formatted, err := c.FormatStream(renamed)
					assert.NoError(t, err)
					assert.Contains(t, formatted, "renamed_target")
					again, err := c.FormatStream(renamed)
					assert.NoError(t, err)
					assert.Equal(t, formatted, again)
					assembled, err := c.AssembleStream(t.Context(), renamed)
					assert.NoError(t, err)
					assert.Equal(t, baseline.Binary, assembled.Binary)
					assert.Equal(t, instructionForms(before), instructionForms(renamed))
					assert.NoError(t, renamed.RenameSymbols(map[string]string{"renamed_entry": "entry", "renamed_target": "target"}))
					assert.Equal(t, before, renamed)
					assert.Equal(t, before, original.stream)
				})
			}
		})
	}
}
