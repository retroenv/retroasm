package assembler

import "errors"

type context struct {
	processNodes bool
	hasElse      bool // to detect invalid multiple else usages

	parent *context
}

// TODO can else have an expression as well?
func processElseCondition(asm *Assembler) error {
	if asm.currentContext.parent == nil {
		return errConditionOutsideIfContext
	}
	if asm.currentContext.hasElse {
		return errors.New("multiple else found")
	}

	asm.currentContext.hasElse = true
	asm.currentContext.processNodes = !asm.currentContext.processNodes
	return nil
}

func processEndifCondition(asm *Assembler) error {
	if asm.currentContext.parent == nil {
		return errConditionOutsideIfContext
	}
	asm.currentContext = asm.currentContext.parent
	return nil
}
