package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/retroenv/retroasm/pkg/arch/chip8"
	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/app"
)

// assembleFile processes the input assembly file and generates output.
func assembleFile(options *optionFlags, args []string) error {
	inputData, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("opening input file '%s': %w", args[0], err)
	}

	compatMode, err := parseCompatMode(options.compat)
	if err != nil {
		return fmt.Errorf("parsing compatibility mode: %w", err)
	}

	ctx := app.Context()

	// Chip-8 uses the direct assembler API as it is not yet supported by the retroasm high-level API.
	if options.cpu == cpuChip8 {
		return assembleChip8File(ctx, inputData, options.output)
	}

	asm := retroasm.New()
	if err := registerArchitectureForCPU(asm, options.cpu, options.z80Profile, compatMode); err != nil {
		return fmt.Errorf("registering architecture '%s': %w", options.cpu, err)
	}

	input := &retroasm.TextInput{
		Source:     bytes.NewReader(inputData),
		SourceName: args[0],
		ConfigFile: options.config,
	}

	output, err := asm.AssembleText(ctx, input)
	if err != nil {
		return fmt.Errorf("assembling input file '%s': %w", args[0], err)
	}

	if err = os.WriteFile(options.output, output.Binary, 0o644); err != nil {
		return fmt.Errorf("writing output file '%s': %w", options.output, err)
	}

	return nil
}

func parseCompatMode(s string) (config.CompatibilityMode, error) {
	if s == "" {
		return config.CompatDefault, nil
	}
	mode, err := config.ParseCompatibilityMode(s)
	if err != nil {
		return config.CompatDefault, fmt.Errorf("parsing compatibility mode: %w", err)
	}
	return mode, nil
}

func assembleChip8File(ctx context.Context, inputData []byte, outputFile string) error {
	cfg := chip8.New()

	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	if err := asm.Process(ctx, bytes.NewReader(inputData)); err != nil {
		return fmt.Errorf("assembling chip8 input: %w", err)
	}

	if err := os.WriteFile(outputFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing output file '%s': %w", outputFile, err)
	}

	return nil
}
