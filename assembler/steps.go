package assembler

type step struct {
	handler       func(*Assembler) error
	errorTemplate string
}

// steps of the assembler to execute, in order
var steps = []step{
	{
		handler:       parseASTNodesStep,
		errorTemplate: "parsing AST nodes",
	},
	{
		handler:       processMacrosStep,
		errorTemplate: "processing macros",
	},
	{
		handler:       evaluateExpressionsStep,
		errorTemplate: "evaluating expressions",
	},
	{
		handler:       updateDataSizesStep,
		errorTemplate: "updating data sizes",
	},
	{
		handler:       assignAddressesStep,
		errorTemplate: "assigning addresses",
	},
	{
		handler:       generateOpcodesStep,
		errorTemplate: "generating opcodes",
	},
	{
		handler:       writeOutputStep,
		errorTemplate: "writing output",
	},
}
