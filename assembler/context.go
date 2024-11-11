package assembler

type context struct {
	processNodes bool
	hasElse      bool // to detect invalid multiple else usages

	parent *context
}

// TODO can else have an expression as well?
func processElseCondition(expEval *expressionEvaluation) error {
	if expEval.currentContext.parent == nil {
		return errConditionOutsideIfContext
	}
	if expEval.currentContext.hasElse {
		return errMultipleElseFound
	}

	expEval.currentContext.hasElse = true
	expEval.currentContext.processNodes = !expEval.currentContext.processNodes
	return nil
}

func processEndifCondition(expEval *expressionEvaluation) error {
	if expEval.currentContext.parent == nil {
		return errConditionOutsideIfContext
	}
	expEval.currentContext = expEval.currentContext.parent
	return nil
}
