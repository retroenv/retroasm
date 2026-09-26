package codec_test

import (
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestCodecRewriteRetainsRemovedMetadataAcrossArchitectures(t *testing.T) {
	for _, test := range streamEquivalenceCases() {
		t.Run(test.name, func(t *testing.T) {
			c := test.newCodec(t)
			parsed, err := c.ParseStream(t.Context(), equivalenceInputSource, strings.NewReader(test.source))
			assert.NoError(t, err)
			baseline, err := c.AssembleStream(t.Context(), parsed)
			assert.NoError(t, err)
			stream := baseline.Stream.Copy()
			before := stream.Copy()
			edits := []ast.EntryEdit{{SourceIndex: ast.NoSourceEntry, Node: &ast.Comment{Message: "inserted"}}}
			for index := 1; index < stream.Len(); index++ {
				edits = append(edits, ast.EntryEdit{SourceIndex: index})
			}
			assert.NoError(t, stream.Rewrite(edits))
			assert.Equal(t, []ast.Entry{before.At(0)}, stream.RemovedEntries())
			assert.Equal(t, ast.SourcePosition{}, stream.At(0).Position)
			assert.NoError(t, c.ValidateStream(stream))
			result, err := c.AssembleStream(t.Context(), stream)
			assert.NoError(t, err)
			assert.Equal(t, baseline.Binary, result.Binary)
			assert.Equal(t, stream.RemovedEntries(), result.Stream.RemovedEntries())
			for index := 1; index < stream.Len(); index++ {
				assert.Equal(t, before.At(index), result.Stream.At(index))
			}
			formatted, err := c.FormatStream(result.Stream)
			assert.NoError(t, err)
			assert.Contains(t, formatted, "inserted")
			assert.False(t, strings.Contains(formatted, "header"))
			again, err := c.AssembleStream(t.Context(), result.Stream)
			assert.NoError(t, err)
			assert.Equal(t, result.Stream, again.Stream)
		})
	}
}
