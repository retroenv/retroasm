package assembler

// Steps of the assembler to execute, in order.
func (asm *Assembler[T]) Steps() []step[T] {
	return []step[T]{
		{
			handler:       processMacrosStep[T],
			errorTemplate: "processing macros",
		},
		{
			handler:       evaluateExpressionsStep[T],
			errorTemplate: "evaluating expressions",
		},
		{
			handler:       updateDataSizesStep[T],
			errorTemplate: "updating data sizes",
		},
		{
			handler:       assignAddressesStep[T],
			errorTemplate: "assigning addresses",
		},
		{
			handler:       generateOpcodesStep[T],
			errorTemplate: "generating opcodes",
		},
		{
			handler:       writeOutputStep[T],
			errorTemplate: "writing output",
		},
	}
}

type step[T any] struct {
	handler       func(*Assembler[T]) error
	errorTemplate string
}
