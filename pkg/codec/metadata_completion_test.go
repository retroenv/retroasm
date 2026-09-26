package codec_test

import (
	"strings"
	"testing"

	asmcpu6502 "github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestCodecCompletesMetadataAfterEntryInsertion(t *testing.T) {
	t.Parallel()
	configuration := asmcpu6502.New()
	configuration.CompatibilityMode = config.CompatCa65
	segment := &config.Segment{
		Memory:      config.Memory{Name: "code", Start: 0, Size: 0x10000},
		SegmentName: "code",
		Align:       16,
	}
	configuration.Segments = map[string]*config.Segment{"code": segment}
	configuration.SegmentsOrdered = []*config.Segment{segment}
	c, err := codec.New(configuration)
	assert.NoError(t, err)
	parsed, err := c.ParseStream(t.Context(), "entries.asm",
		strings.NewReader(".segment \"code\"\nentry:\n.addr entry, entry + 1"))
	assert.NoError(t, err)

	stream := ast.NewStream(parsed.Entries()...)
	stream.RecordSegmentChange(parsed.SegmentChanges()[0])
	stream.RecordRelocation(parsed.Relocations()[0])
	assert.NoError(t, stream.Replace(0, 0, []ast.Entry{parsed.At(0)}))
	stream.Append(parsed.At(2))
	before := stream.Copy()
	assembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	assert.Equal(t, before, stream)
	assert.Equal(t, []byte{0, 0, 1, 0, 0, 0, 1, 0}, assembly.Binary)
	assert.Len(t, assembly.Stream.SegmentChanges(), 2)
	assert.Len(t, assembly.Stream.Relocations(), 4)
	assert.Equal(t, before.Entries(), assembly.Stream.Entries())
	assert.NoError(t, assembly.Stream.Validate())

	repeated, err := c.AssembleStream(t.Context(), assembly.Stream)
	assert.NoError(t, err)
	assert.Equal(t, assembly.Binary, repeated.Binary)
	assert.Equal(t, assembly.Stream, repeated.Stream)
}
