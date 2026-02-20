package number

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Sentinel errors for number parsing operations.
var (
	ErrInvalidNumberBaseCombination = errors.New("invalid number base combination")
	ErrInvalidBinaryChar            = errors.New("invalid binary character")
	ErrInvalidHexChar               = errors.New("invalid hex character")
	ErrInvalidNumberChar            = errors.New("invalid number character")
	ErrNumberExceedsWidth           = errors.New("number exceeds data width")
	ErrUnsupportedDataWidth         = errors.New("unsupported data width")
	ErrParseNumber                  = errors.New("failed to parse number")
)

const (
	unsupportedDataWidthMsg = "unsupported data byte width %d"
	numberExceedsMsg        = "number %d exceeds %d byte"
)

// Parse parses a number string and returns it as uint64.
// Supports multiple formats: decimal (123), hex ($FF, 0xFF), binary (%1010, 1010b), immediate (#123).
func Parse(value string) (uint64, error) {
	var base, idx int
	builder := &strings.Builder{}
	builder.Grow(len(value)) // Pre-allocate for performance

	if len(value) > 0 && value[0] == '#' {
		idx++
	}

	for i := range len(value) {
		if idx >= len(value) {
			break
		}
		c := rune(value[idx])
		c = unicode.ToLower(c)

		if err := parseCharacter(c, &i, &idx, &base, value, builder); err != nil {
			return 0, err
		}

		idx++
	}

	s := builder.String()
	i, err := strconv.ParseUint(s, base, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: decoding string '%s': %w", ErrParseNumber, s, err)
	}

	return i, nil
}

// ParseToBytes parses a number string to a byte buffer of specific byte width.
// This is useful for parsing a word string into a byte array of 2 bytes.
func ParseToBytes(value string, dataWidth int) ([]byte, error) {
	i, err := Parse(value)
	if err != nil {
		return nil, err
	}

	if err := CheckDataWidth(i, dataWidth); err != nil {
		return nil, err
	}

	return WriteToBytes(i, dataWidth)
}

// CheckDataWidth verifies that the given int fits into the expected data byte width.
func CheckDataWidth(i uint64, dataWidth int) error {
	switch dataWidth {
	case 1:
		if i > math.MaxUint8 {
			return fmt.Errorf("%w: "+numberExceedsMsg, ErrNumberExceedsWidth, i, 1)
		}
	case 2:
		if i > math.MaxUint16 {
			return fmt.Errorf("%w: "+numberExceedsMsg, ErrNumberExceedsWidth, i, 2)
		}
	case 4:
		if i > math.MaxUint32 {
			return fmt.Errorf("%w: "+numberExceedsMsg, ErrNumberExceedsWidth, i, 4)
		}
	case 8:

	default:
		return fmt.Errorf("%w: "+unsupportedDataWidthMsg, ErrUnsupportedDataWidth, dataWidth)
	}

	return nil
}

// WriteToBytes writes a number to a byte buffer of specific data byte width.
func WriteToBytes(i uint64, dataWidth int) ([]byte, error) {
	switch dataWidth {
	case 1:
		return []byte{uint8(i)}, nil
	case 2:
		data := make([]byte, 2)
		binary.LittleEndian.PutUint16(data, uint16(i))
		return data, nil
	case 4:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(i))
		return data, nil
	case 8:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, i)
		return data, nil
	default:
		return nil, fmt.Errorf("%w: "+unsupportedDataWidthMsg, ErrUnsupportedDataWidth, dataWidth)
	}
}

func parseCharacter(r rune, i, idx, base *int, value string, builder *strings.Builder) error {
	if handled, err := parseBaseIndicator(r, i, idx, base, value); handled || err != nil {
		return err
	}

	return parseDigit(r, *base, builder)
}

// parseBaseIndicator handles base prefix and suffix detection for number parsing.
// Returns true if the rune was consumed as a base indicator.
func parseBaseIndicator(r rune, i, idx, base *int, value string) (bool, error) {
	switch {
	case r == '%':
		if *base > 0 {
			return false, ErrInvalidNumberBaseCombination
		}
		*base = 2
		return true, nil

	case r == 'b' && *base == 0 && *i+1 == len(value):
		*base = 2
		return true, nil

	case r == '$':
		if *base > 0 {
			return false, ErrInvalidNumberBaseCombination
		}
		*base = 16
		return true, nil

	case r == '0' && *i == 0 && len(value) > *idx+1 && (value[*idx+1] == 'x' || value[*idx+1] == 'X'):
		*base = 16
		*idx++
		*i++
		return true, nil

	default:
		return false, nil
	}
}

func parseDigit(r rune, base int, builder *strings.Builder) error {
	switch {
	case unicode.IsDigit(r):
		if base == 2 && r > '1' {
			return fmt.Errorf("%w: '%c'", ErrInvalidBinaryChar, r)
		}
		builder.WriteRune(r)

	case r >= 'a' && r <= 'f':
		if base != 0 && base != 16 {
			return fmt.Errorf("%w: '%c'", ErrInvalidHexChar, r)
		}
		builder.WriteRune(r)

	default:
		return fmt.Errorf("%w: '%c'", ErrInvalidNumberChar, r)
	}

	return nil
}
