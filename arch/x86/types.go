package x86

// AddressingMode represents x86 addressing modes.
type AddressingMode int

const (
	// Register addressing modes.
	RegisterAddressing AddressingMode = iota
	ImmediateAddressing

	// Memory addressing modes.
	DirectAddressing       // [1234h]
	RegisterIndirect       // [BX], [SI], [DI], [BP]
	BasedAddressing        // [BX+disp], [BP+disp]
	IndexedAddressing      // [SI+disp], [DI+disp]
	BasedIndexedAddressing // [BX+SI], [BP+DI], etc.
)

// RegisterSize represents the size of x86 registers.
type RegisterSize int

const (
	Byte RegisterSize = iota // 8-bit
	Word                     // 16-bit
)

// AddressingInfo contains information about an instruction's addressing mode.
type AddressingInfo struct {
	Opcode byte
	Size   int  // Total instruction size in bytes
	HasRM  bool // Whether instruction uses ModR/M byte
}

// Instruction represents an x86 instruction definition.
type Instruction struct {
	Name       string
	Addressing map[AddressingMode]AddressingInfo
}

// HasAddressing returns true if the instruction supports the given addressing mode.
func (i *Instruction) HasAddressing(mode AddressingMode) bool {
	_, ok := i.Addressing[mode]
	return ok
}
