package number

import (
	"encoding/binary"
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestWriteToBytesWithOrder(t *testing.T) {
	tests := []struct {
		name     string
		value    uint64
		width    int
		order    binary.ByteOrder
		expected []byte
	}{
		{name: "little word", value: 0x1234, width: 2, order: binary.LittleEndian, expected: []byte{0x34, 0x12}},
		{name: "big byte", value: 0x12, width: 1, order: binary.BigEndian, expected: []byte{0x12}},
		{name: "big word", value: 0x1234, width: 2, order: binary.BigEndian, expected: []byte{0x12, 0x34}},
		{name: "big long", value: 0x123456, width: 3, order: binary.BigEndian, expected: []byte{0x12, 0x34, 0x56}},
		{name: "big double word", value: 0x12345678, width: 4, order: binary.BigEndian, expected: []byte{0x12, 0x34, 0x56, 0x78}},
		{name: "big quad word", value: 0x123456789abcdef0, width: 8, order: binary.BigEndian, expected: []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := WriteToBytesWithOrder(test.value, test.width, test.order)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}

	_, err := WriteToBytesWithOrder(1, 1, nil)
	assert.ErrorIs(t, err, ErrUnsupportedByteOrder)
}

func TestNumberParseToBytes(t *testing.T) {
	tests := []struct {
		input         string
		dataByteWidth int
		expected      []byte
		expectedErr   error
	}{
		{input: "0x12", dataByteWidth: 1, expected: []byte{0x12}},
		{input: "0x1234", dataByteWidth: 2, expected: []byte{0x34, 0x12}},
		{input: "0x123456", dataByteWidth: 3, expected: []byte{0x56, 0x34, 0x12}},
		{input: "0x12345678", dataByteWidth: 4, expected: []byte{0x78, 0x56, 0x34, 0x12}},
		{input: "0x123456789abcdef0", dataByteWidth: 8, expected: []byte{0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12}},
		{input: "0xx12", dataByteWidth: 1, expectedErr: ErrInvalidNumberChar},
		{input: "0x12", dataByteWidth: 0, expectedErr: ErrUnsupportedDataWidth},
		{input: "0x123", dataByteWidth: 1, expectedErr: ErrNumberExceedsWidth},
		{input: "0x12345", dataByteWidth: 2, expectedErr: ErrNumberExceedsWidth},
		{input: "0x1234567", dataByteWidth: 3, expectedErr: ErrNumberExceedsWidth},
		{input: "0x123456789", dataByteWidth: 4, expectedErr: ErrNumberExceedsWidth},
	}

	for _, tt := range tests {
		b, err := ParseToBytes(tt.input, tt.dataByteWidth)

		if tt.expectedErr != nil {
			assert.ErrorIs(t, err, tt.expectedErr)
		} else {
			assert.NoError(t, err, "input: "+tt.input)
			assert.Equal(t, tt.expected, b)
		}
	}
}

func TestNumberParse(t *testing.T) {
	tests := []struct {
		input       string
		expected    uint64
		expectedErr error
	}{
		{input: "0", expected: 0},
		{input: "$ABCD", expected: 0xABCD},
		{input: "12345", expected: 12345},
		{input: "%01010101", expected: 85},
		{input: "01010101b", expected: 85},
		{input: "#%10001000", expected: 136},
		{input: "0xab", expected: 0xab},
		{input: "0ABhCDh", expectedErr: ErrInvalidNumberChar},
		{input: "$0ABCDh", expectedErr: ErrInvalidNumberChar},
		{input: "%01010101b", expectedErr: ErrInvalidHexChar},
		{input: "%%1", expectedErr: ErrInvalidNumberBaseCombination},
		{input: "$0x1", expectedErr: ErrInvalidNumberChar},
		{input: "0x12345678901234567890", expectedErr: ErrParseNumber},
		{input: "%2", expectedErr: ErrInvalidBinaryChar},
	}

	for _, tt := range tests {
		i, err := Parse(tt.input)

		if tt.expectedErr != nil {
			assert.ErrorIs(t, err, tt.expectedErr)
		} else {
			assert.NoError(t, err, "input: "+tt.input)
			assert.Equal(t, tt.expected, i)
		}
	}
}
