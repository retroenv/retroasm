// Package main implements retroasm, a retro computer assembler.
// It provides command-line interface for assembling retro computer code,
// currently supporting NES/6502 architecture with ca65-compatible configuration.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/app"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/buildinfo"
	"github.com/retroenv/retrogolib/log"
)

// Structured errors for validation.
var (
	ErrUnsupportedSystem = errors.New("unsupported system")
	ErrUnsupportedCPU    = errors.New("unsupported CPU architecture")
	ErrIncompatibleArch  = errors.New("incompatible system and CPU combination")
)

// Build-time variables set by go build -ldflags.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// optionFlags holds command-line options and runtime configuration.
type optionFlags struct {
	logger *log.Logger

	// File paths
	config string
	output string

	// Architecture configuration
	cpu    string
	system string

	// Logging configuration
	debug bool
	quiet bool
}

func main() {
	options, args := readArguments()
	if !options.quiet {
		printBanner(options)
	}

	logFields := buildLogFields(args[0], options)
	options.logger.Info("Assembling file...", logFields...)

	if err := assembleFile(options, args); err != nil {
		options.logger.Error("Assembling failed", log.Err(err))
		os.Exit(1)
	}

	options.logger.Info("Assembling finished successfully", log.String("output", options.output))
}

// buildLogFields creates log fields for assembly operation.
func buildLogFields(input string, options *optionFlags) []log.Field {
	fields := []log.Field{log.String("input", input)}
	if options.cpu != "" {
		fields = append(fields, log.String("cpu", options.cpu))
	}
	if options.system != "" {
		fields = append(fields, log.String("system", options.system))
	}
	return fields
}

// createLogger creates a configured logger based on options.
func createLogger(options *optionFlags) *log.Logger {
	cfg := log.DefaultConfig()
	if options.debug {
		cfg.Level = log.DebugLevel
	}
	if options.quiet {
		cfg.Level = log.ErrorLevel
	}
	return log.NewWithConfig(cfg)
}

// Supported architectures and systems.
const (
	supportedCPU    = "6502"
	supportedSystem = "nes"
)

// validateAndProcessArchitecture validates the CPU and system flags and applies defaults.
// Currently only supports NES system and 6502 CPU architecture.
// If system is "nes", it defaults to 6502 CPU if no CPU is specified.
func validateAndProcessArchitecture(options *optionFlags) error {
	// Validate system if specified
	if err := validateSystem(options); err != nil {
		return err
	}

	// Validate CPU if specified
	if err := validateCPU(options); err != nil {
		return err
	}

	// Validate compatibility between system and CPU
	if options.system == supportedSystem && options.cpu != "" && options.cpu != supportedCPU {
		return fmt.Errorf("%w: NES system requires 6502 CPU architecture, got: %s", ErrIncompatibleArch, options.cpu)
	}

	return nil
}

func validateSystem(options *optionFlags) error {
	if options.system == "" {
		return nil
	}

	sys, ok := arch.SystemFromString(options.system)
	if !ok {
		return fmt.Errorf("%w: %s (only '%s' is currently supported)", ErrUnsupportedSystem, options.system, supportedSystem)
	}

	// Currently only support NES system
	if sys != arch.NES {
		return fmt.Errorf("%w: %s (only '%s' is currently supported)", ErrUnsupportedSystem, sys, supportedSystem)
	}

	// If system is NES and no CPU specified, default to 6502
	if sys == arch.NES && options.cpu == "" {
		options.cpu = supportedCPU
		if options.debug {
			options.logger.Debug("Defaulting to 6502 CPU for NES system")
		}
	}

	return nil
}

func validateCPU(options *optionFlags) error {
	if options.cpu == "" {
		return nil
	}

	cpu, ok := arch.FromString(options.cpu)
	if !ok {
		return fmt.Errorf("%w: %s (only '%s' is currently supported)", ErrUnsupportedCPU, options.cpu, supportedCPU)
	}

	// Currently only support 6502 CPU
	if cpu != arch.M6502 {
		return fmt.Errorf("%w: %s (only '%s' is currently supported)", ErrUnsupportedCPU, cpu, supportedCPU)
	}

	return nil
}

// readArguments parses command-line arguments and validates configuration.
func readArguments() (*optionFlags, []string) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	options := &optionFlags{}

	// Define command-line flags
	flags.BoolVar(&options.debug, "debug", false, "enable debug logging")
	flags.StringVar(&options.config, "c", "", "assembler config file")
	flags.StringVar(&options.output, "o", "", "name of the output file")
	flags.StringVar(&options.cpu, "cpu", "6502", "target CPU architecture (6502)")
	flags.StringVar(&options.system, "system", "nes", "target system (nes)")
	flags.BoolVar(&options.quiet, "q", false, "perform operations quietly")

	err := flags.Parse(os.Args[1:])
	args := flags.Args()

	// Create logger early for error reporting
	logger := createLogger(options)
	options.logger = logger

	// Validate required arguments
	if err != nil || len(args) == 0 || options.output == "" {
		showUsageAndExit(options, flags)
	}

	// Validate and process architecture configuration
	if err := validateAndProcessArchitecture(options); err != nil {
		logger.Error("Invalid architecture configuration", log.Err(err))
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return options, args
}

// showUsageAndExit displays usage information and exits.
func showUsageAndExit(options *optionFlags, flags *flag.FlagSet) {
	printBanner(options)
	fmt.Printf("usage: retroasm [options] <file to assemble>\n\n")
	flags.PrintDefaults()
	fmt.Println()
	os.Exit(1)
}

// printBanner displays the application banner.
func printBanner(options *optionFlags) {
	if !options.quiet {
		fmt.Println("[-------------------------------------]")
		fmt.Println("[ retroasm - retro computer assembler ]")
		fmt.Printf("[-------------------------------------]\n\n")
		options.logger.Info("Build info", log.String("version", buildinfo.Version(version, commit, date)))
	}
}

// assembleFile processes the input assembly file and generates output.
func assembleFile(options *optionFlags, args []string) error {
	// Create assembler and register architecture
	asm := retroasm.New()
	m6502Arch := m6502.New()
	adapter := retroasm.NewArchitectureAdapter(string(arch.M6502), m6502Arch, m6502Arch)
	if err := asm.RegisterArchitecture(string(arch.M6502), adapter); err != nil {
		return fmt.Errorf("registering architecture: %w", err)
	}

	// Read input assembly file
	inputData, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("opening input file '%s': %w", args[0], err)
	}

	// Assemble using text input
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

	// Write output file with appropriate permissions
	if err = os.WriteFile(options.output, output.Binary, 0o644); err != nil {
		return fmt.Errorf("writing output file '%s': %w", options.output, err)
	}

	return nil
}
