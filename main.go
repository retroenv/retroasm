// Package main implements a retro computer assembler.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/retroenv/retroasm/arch/m6502"
	"github.com/retroenv/retroasm/assembler"
	"github.com/retroenv/retrogolib/buildinfo"
	"github.com/retroenv/retrogolib/log"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

type optionFlags struct {
	logger *log.Logger

	config string
	output string

	debug bool
	quiet bool
}

func main() {
	options, args := readArguments()
	if !options.quiet {
		printBanner(options)
	}

	options.logger.Info("Assembling file...", log.String("input", args[0]))
	if err := assembleFile(options, args); err != nil {
		options.logger.Error("Assembling failed", log.Err(err))
	}
	options.logger.Info("Assembling finished successfully", log.String("output", options.output))
}

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

func readArguments() (*optionFlags, []string) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	options := &optionFlags{}

	flags.BoolVar(&options.debug, "debug", false, "enable debug logging")
	flags.StringVar(&options.config, "c", "", "assembler config file")
	flags.StringVar(&options.output, "o", "", "name of the output file")
	flags.BoolVar(&options.quiet, "q", false, "perform operations quietly")

	err := flags.Parse(os.Args[1:])
	args := flags.Args()

	logger := createLogger(options)
	options.logger = logger

	if err != nil || (len(args) == 0) || options.output == "" {
		printBanner(options)
		fmt.Printf("usage: retroasm [options] <file to assemble>\n\n")
		flags.PrintDefaults()
		fmt.Println()
		os.Exit(1)
	}

	return options, args
}

func printBanner(options *optionFlags) {
	if !options.quiet {
		fmt.Println("[-------------------------------------]")
		fmt.Println("[ retroasm - retro computer assembler ]")
		fmt.Printf("[-------------------------------------]\n\n")
		options.logger.Info("Build info", log.String("version", buildinfo.Version(version, commit, date)))
	}
}

func assembleFile(options *optionFlags, args []string) error {
	cfg := m6502.New()
	if options.config != "" {
		cfgData, err := os.ReadFile(options.config)
		if err != nil {
			return fmt.Errorf("opening config file '%s': %w", options.config, err)
		}

		if err := cfg.ReadCa65Config(bytes.NewReader(cfgData)); err != nil {
			return fmt.Errorf("reading config file '%s': %w", options.config, err)
		}
	}

	input, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("opening input file '%s': %w", args[0], err)
	}

	var buf bytes.Buffer
	asm := assembler.New(cfg, bytes.NewReader(input), &buf)

	if err = asm.Process(); err != nil {
		return fmt.Errorf("assembling input file '%s': %w", args[0], err)
	}
	if err = os.WriteFile(options.output, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("writing output file '%s': %w", options.output, err)
	}
	return nil
}
