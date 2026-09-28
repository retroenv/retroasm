package assembler

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retrogolib/assert"
)

func TestMemoryWriteBounds(t *testing.T) {
	for _, fill := range []bool{false, true} {
		mem := newMemory(config.Memory{
			Start:     0xc000,
			Size:      4,
			Fill:      fill,
			FillValue: 0xff,
		})
		assert.NoError(t, mem.write([]byte{1, 2, 3, 4}, 0xc000))
		assert.NoError(t, mem.write(nil, 0xc004))
		for _, address := range []uint64{0xbfff, 0xc004, 0xffffffffffffffff} {
			assert.Error(t, mem.write([]byte{5}, address))
		}
		assert.Error(t, mem.write([]byte{5, 6}, 0xc003))
		assert.Equal(t, []byte{1, 2, 3, 4}, mem.data)
	}
}

func TestMemoryWritePreservesFillAndRelativeOffsets(t *testing.T) {
	mem := newMemory(config.Memory{
		Start:     0x8000,
		Size:      4,
		Fill:      true,
		FillValue: 0xa5,
	})
	assert.Equal(t, []byte{0xa5, 0xa5, 0xa5, 0xa5}, mem.data)
	assert.NoError(t, mem.write([]byte{7}, 0x8001))
	assert.Equal(t, []byte{0xa5, 7, 0xa5, 0xa5}, mem.data)
	mem = newMemory(config.Memory{
		Start: 0x8000,
		Size:  4,
	})
	assert.NoError(t, mem.write([]byte{7}, 0x8001))
	assert.Equal(t, []byte{0, 7}, mem.data)
}
