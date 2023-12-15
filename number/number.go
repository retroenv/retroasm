// Package number provides number string parsing helper.
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

var errInvalidNumberBaseCombination = errors.New("invalid number base combination")

// Parse a number string and return it as uint64.
func Parse(value string) (uint64, error) {
	var base, idx int
	builder := &strings.Builder{}

	if len(value) > 0 && value[0] == '#' {
		idx++
	}

	for i := 0; idx < len(value); i++ {
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
		return 0, fmt.Errorf("decoding string '%s': %w", s, err)
	}

	return i, nil
}

// nolint: gocognit,cyclop
func parseCharacter(r rune, i, idx, base *int, value string, builder *strings.Builder) error {
	switch {
	case r == '%': // prefix
		if *base > 0 {
			return errInvalidNumberBaseCombination
		}
		*base = 2 // binary

	case r == 'b' && *base == 0 && *i+1 == len(value): // suffix
		if *base > 0 {
			return errInvalidNumberBaseCombination
		}
		*base = 2 // binary

	case r == '$':
		if *base > 0 {
			return errInvalidNumberBaseCombination
		}
		*base = 16 // hex

	case r == '0' && *i == 0 && len(value) > *idx+1 && (value[*idx+1] == 'x' || value[*idx+1] == 'X'):
		*base = 16 // hex
		*idx++
		*i++

	case unicode.IsDigit(r):
		if *base == 2 && r > '1' {
			return fmt.Errorf("invalid binary character '%c'", r)
		}
		builder.WriteRune(r)

	case r >= 'a' && r <= 'f':
		if *base != 0 && *base != 16 {
			return fmt.Errorf("invalid hex character '%c'", r)
		}
		builder.WriteRune(r)

	default:
		return fmt.Errorf("invalid number character '%c'", r)
	}

	return nil
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
			return fmt.Errorf("number %d exceeds 1 byte", i)
		}
	case 2:
		if i > math.MaxUint16 {
			return fmt.Errorf("number %d exceeds 2 byte", i)
		}
	case 4:
		if i > math.MaxUint32 {
			return fmt.Errorf("number %d exceeds 4 byte", i)
		}
	case 8:

	default:
		return fmt.Errorf("unsupported data byte width %d", dataWidth)
	}

	return nil
}

// WriteToBytes writes a number to a byte buffer of specific data byte width.
func WriteToBytes(i uint64, dataWidth int) ([]byte, error) {
	data := make([]byte, dataWidth)

	switch dataWidth {
	case 1:
		data = []byte{uint8(i)}
	case 2:
		binary.LittleEndian.PutUint16(data, uint16(i))
	case 4:
		binary.LittleEndian.PutUint32(data, uint32(i))
	case 8:
		binary.LittleEndian.PutUint64(data, i)
	default:
		return nil, fmt.Errorf("unsupported data byte width %d", dataWidth)
	}

	return data, nil
}
