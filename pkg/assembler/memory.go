package assembler

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/assembler/config"
)

// memory is a memory segment of the output file.
type memory struct {
	start uint64
	size  uint64
	data  []byte
}

// newMemory creates a new memory instance with the given configuration.
func newMemory(cfg config.Memory) *memory {
	o := &memory{
		start: cfg.Start,
		size:  cfg.Size,
	}

	if cfg.Fill {
		o.data = make([]byte, cfg.Size)
		for i := range cfg.Size {
			o.data[i] = cfg.FillValue
		}
	}

	return o
}

// write stores bytes at their address relative to the memory area.
func (o *memory) write(data []byte, address uint64) error {
	if address < o.start || address-o.start > o.size || uint64(len(data)) > o.size-(address-o.start) {
		return fmt.Errorf("write of %d bytes at $%x exceeds memory range $%x with size %d",
			len(data), address, o.start, o.size)
	}
	index := int(address - o.start)

	extendBuf := index - len(o.data) + len(data)
	if extendBuf > 0 {
		b := make([]byte, extendBuf)
		o.data = append(o.data, b...)
	}

	copy(o.data[index:index+len(data)], data)
	return nil
}
