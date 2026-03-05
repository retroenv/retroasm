package profile

import (
	"errors"
	"fmt"
	"strings"

	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/set"
)

// Kind identifies a Z80 instruction profile.
type Kind int

const (
	Default Kind = iota
	StrictDocumented
	GameBoySubset
)

const (
	defaultName          = "default"
	strictDocumentedName = "strict-documented"
	gameBoySubsetName    = "gameboy-z80-subset"
)

var (
	ErrUnsupportedProfile     = errors.New("unsupported z80 profile")
	ErrUnsupportedInstruction = errors.New("instruction is not supported by selected z80 profile")
)

var kindByName = map[string]Kind{
	defaultName:          Default,
	strictDocumentedName: StrictDocumented,
	gameBoySubsetName:    GameBoySubset,
}

var unsupportedGameBoyPrefixes = set.NewFromSlice([]byte{
	cpuz80.PrefixDD,
	cpuz80.PrefixED,
	cpuz80.PrefixFD,
})

var unsupportedGameBoyMnemonics = set.NewFromSlice([]string{
	cpuz80.DjnzName,
	cpuz80.ExName,
	cpuz80.ExxName,
	cpuz80.InName,
	cpuz80.OutName,
})

var undocumentedOpcodeKeys = set.NewFromSlice([]uint16{
	opcodeKey(cpuz80.PrefixED, 0x4C),
	opcodeKey(cpuz80.PrefixED, 0x54),
	opcodeKey(cpuz80.PrefixED, 0x5C),
	opcodeKey(cpuz80.PrefixED, 0x64),
	opcodeKey(cpuz80.PrefixED, 0x6C),
	opcodeKey(cpuz80.PrefixED, 0x74),
	opcodeKey(cpuz80.PrefixED, 0x7C),
	opcodeKey(cpuz80.PrefixED, 0x55),
	opcodeKey(cpuz80.PrefixED, 0x65),
	opcodeKey(cpuz80.PrefixED, 0x75),
	opcodeKey(cpuz80.PrefixED, 0x66),
	opcodeKey(cpuz80.PrefixED, 0x76),
	opcodeKey(cpuz80.PrefixED, 0x7E),
})

func (k Kind) String() string {
	switch k {
	case StrictDocumented:
		return strictDocumentedName
	case GameBoySubset:
		return gameBoySubsetName
	default:
		return defaultName
	}
}

// Parse converts a CLI/profile string into a profile kind.
func Parse(value string) (Kind, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return Default, nil
	}

	kind, ok := kindByName[normalized]
	if !ok {
		return Default, fmt.Errorf(
			"%w: %s (supported: %s, %s, %s)",
			ErrUnsupportedProfile,
			value,
			defaultName,
			strictDocumentedName,
			gameBoySubsetName,
		)
	}

	return kind, nil
}

// ValidateInstruction checks if the selected instruction is allowed by the profile.
func ValidateInstruction(
	kind Kind,
	instruction *cpuz80.Instruction,
	addressing cpuz80.AddressingMode,
	registerParams []cpuz80.RegisterParam,
) error {

	if kind == Default {
		return nil
	}
	if instruction == nil {
		return fmt.Errorf("%w: missing instruction details", ErrUnsupportedInstruction)
	}

	info, err := opcodeInfo(instruction, addressing, registerParams)
	if err != nil {
		return fmt.Errorf("resolving opcode info: %w", err)
	}

	if isUndocumentedInstruction(instruction, info) {
		return fmt.Errorf(
			"%w: instruction '%s' (%s) is undocumented for profile '%s'",
			ErrUnsupportedInstruction,
			instruction.Name,
			opcodeDescription(info),
			kind.String(),
		)
	}

	if kind != GameBoySubset {
		return nil
	}

	if unsupportedGameBoyMnemonics.Contains(instruction.Name) {
		return fmt.Errorf(
			"%w: instruction '%s' is outside profile '%s'",
			ErrUnsupportedInstruction,
			instruction.Name,
			kind.String(),
		)
	}

	if unsupportedGameBoyPrefixes.Contains(info.Prefix) {
		return fmt.Errorf(
			"%w: instruction '%s' uses unsupported prefix 0x%02X for profile '%s'",
			ErrUnsupportedInstruction,
			instruction.Name,
			info.Prefix,
			kind.String(),
		)
	}

	return nil
}

func isUndocumentedInstruction(instruction *cpuz80.Instruction, info cpuz80.OpcodeInfo) bool {
	if instruction.Unofficial {
		return true
	}

	switch instruction.Name {
	case cpuz80.SllName, cpuz80.InfName, cpuz80.OutfName:
		return true
	}

	if info.Prefix == cpuz80.PrefixCB && info.Opcode >= 0x30 && info.Opcode <= 0x37 {
		return true
	}

	return undocumentedOpcodeKeys.Contains(opcodeKey(info.Prefix, info.Opcode))
}

func opcodeInfo(
	instruction *cpuz80.Instruction,
	addressing cpuz80.AddressingMode,
	registerParams []cpuz80.RegisterParam,
) (cpuz80.OpcodeInfo, error) {

	switch len(registerParams) {
	case 1:
		info, ok := instruction.RegisterOpcodes[registerParams[0]]
		if ok {
			return info, nil
		}
	case 2:
		key := [2]cpuz80.RegisterParam{registerParams[0], registerParams[1]}
		info, ok := instruction.RegisterPairOpcodes[key]
		if ok {
			return info, nil
		}
	}

	info, ok := instruction.Addressing[addressing]
	if ok {
		return info, nil
	}

	if len(instruction.Addressing) == 1 {
		for _, opcodeInfo := range instruction.Addressing {
			return opcodeInfo, nil
		}
	}

	return cpuz80.OpcodeInfo{}, fmt.Errorf(
		"%w: no opcode for instruction '%s' with addressing %d and %d register params",
		ErrUnsupportedInstruction,
		instruction.Name,
		addressing,
		len(registerParams),
	)
}

func opcodeDescription(info cpuz80.OpcodeInfo) string {
	if info.Prefix == 0 {
		return fmt.Sprintf("opcode 0x%02X", info.Opcode)
	}

	return fmt.Sprintf("prefix 0x%02X opcode 0x%02X", info.Prefix, info.Opcode)
}

func opcodeKey(prefix, opcode byte) uint16 {
	return uint16(prefix)<<8 | uint16(opcode)
}
