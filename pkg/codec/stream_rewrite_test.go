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

func TestCodecNodeEditAcrossArchitectures(t *testing.T) {
	for _, test := range streamEquivalenceCases() {
		t.Run(test.name, func(t *testing.T) {
			c := test.newCodec(t)
			parsed, err := c.ParseStream(t.Context(), equivalenceInputSource, strings.NewReader(test.source))
			assert.NoError(t, err)
			baseline, err := c.AssembleStream(t.Context(), parsed)
			assert.NoError(t, err)
			stream := baseline.Stream.Copy()
			edit, err := stream.EditNodes()
			assert.NoError(t, err)
			for index := range edit.Len() {
				assert.True(t, ast.Equal(stream.At(index).Node, edit.At(index)))
			}
			nodes := make([]ast.Node, 0, edit.Len()+2)
			nodes = append(nodes, &ast.Comment{Message: "inserted"})
			nodes = append(nodes, edit.Nodes()[1:]...)
			nodes = append(nodes, edit.At(0), edit.At(0).Copy())
			assert.NoError(t, edit.Commit(nodes))
			assert.Equal(t, ast.SourcePosition{}, stream.At(0).Position)
			assert.Equal(t, baseline.Stream.At(0), stream.At(stream.Len()-1))
			assert.Equal(t, baseline.Stream.At(0), stream.At(stream.Len()-2))
			assert.NoError(t, c.ValidateStream(stream))
			assembled, err := c.AssembleStream(t.Context(), stream)
			assert.NoError(t, err)
			assert.Equal(t, baseline.Binary, assembled.Binary)
			assert.Equal(t, stream.Entries(), assembled.Stream.Entries())
			again, err := c.AssembleStream(t.Context(), assembled.Stream)
			assert.NoError(t, err)
			assert.Equal(t, assembled.Stream, again.Stream)
		})
	}
}
