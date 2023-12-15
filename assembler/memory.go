package assembler

import "github.com/retroenv/assembler/assembler/config"

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
		for i := uint64(0); i < cfg.Size; i++ {
			o.data[i] = cfg.FillValue
		}
	}

	return o
}

// write data using the offset address into the memory, the index will be calculated based on
// the start address of the memory. If the memory config does not specify the fill flag,
// the memory can not be preallocated but has to be written incrementally.
func (o *memory) write(data []byte, offsetAddress uint64) {
	index := int(offsetAddress - o.start)

	extendBuf := index - len(o.data) + len(data)
	if extendBuf > 0 {
		b := make([]byte, extendBuf)
		o.data = append(o.data, b...)
	}

	copy(o.data[index:index+len(data)], data)
}
