package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/app"
)

// assembleFile processes the input assembly file and generates output.
func assembleFile(options *optionFlags, args []string) error {
	asm := retroasm.New()

	if err := registerArchitectureForCPU(asm, options.cpu); err != nil {
		return err
	}

	inputData, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("opening input file '%s': %w", args[0], err)
	}

	input := &retroasm.TextInput{
		Source:     bytes.NewReader(inputData),
		SourceName: args[0],
		ConfigFile: options.config,
	}

	ctx := app.Context()
	output, err := asm.AssembleText(ctx, input)
	if err != nil {
		return fmt.Errorf("assembling input file '%s': %w", args[0], err)
	}

	if err = os.WriteFile(options.output, output.Binary, 0o644); err != nil {
		return fmt.Errorf("writing output file '%s': %w", options.output, err)
	}

	return nil
}
