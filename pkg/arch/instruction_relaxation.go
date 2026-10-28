package arch

// InstructionRelaxer permits target encoders to grow instructions after all
// symbols have addresses. It may grow each instruction once and must retain
// that form on later layout passes.
// The assembler repeats address assignment until no instruction grows.
type InstructionRelaxer interface {
	RelaxInstruction(AddressAssigner, Instruction) (bool, error)
}
