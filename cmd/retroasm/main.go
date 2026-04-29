// Package main implements retroasm, a retro computer assembler.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/retroenv/retrogolib/buildinfo"
	"github.com/retroenv/retrogolib/log"
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
	config string
	output string
	cpu    string
	system string
	debug  bool
	quiet  bool
}

func main() {
	options, args := readArguments()
	printBanner(options)

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

// readArguments parses command-line arguments and validates configuration.
func readArguments() (*optionFlags, []string) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	options := &optionFlags{}

	flags.BoolVar(&options.debug, "debug", false, "enable debug logging")
	flags.StringVar(&options.config, "c", "", "assembler config file")
	flags.StringVar(&options.output, "o", "", "name of the output file")
	flags.StringVar(&options.cpu, "cpu", "", "target CPU architecture (6502, chip8, z80)")
	flags.StringVar(&options.system, "system", "", "target system (nes, chip8, generic, gameboy, zx-spectrum)")
	flags.BoolVar(&options.quiet, "q", false, "perform operations quietly")

	err := flags.Parse(os.Args[1:])
	args := flags.Args()

	logger := createLogger(options)
	options.logger = logger

	if err != nil || len(args) == 0 || options.output == "" {
		showUsageAndExit(options, flags)
	}

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
