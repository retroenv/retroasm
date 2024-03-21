package number

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestNumberParseToBytes(t *testing.T) {
	tests := []struct {
		input         string
		dataByteWidth int
		expected      []byte
		expectedErr   bool
	}{
		{input: "0x12", dataByteWidth: 1, expected: []byte{0x12}},
		{input: "0x1234", dataByteWidth: 2, expected: []byte{0x34, 0x12}},
		{input: "0x12345678", dataByteWidth: 4, expected: []byte{0x78, 0x56, 0x34, 0x12}},
		{input: "0x123456789abcdef0", dataByteWidth: 8, expected: []byte{0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12}},
		{input: "0xx12", dataByteWidth: 1, expectedErr: true},
		{input: "0x12", dataByteWidth: 0, expectedErr: true},
		{input: "0x123", dataByteWidth: 1, expectedErr: true},
		{input: "0x12345", dataByteWidth: 2, expectedErr: true},
		{input: "0x123456789", dataByteWidth: 4, expectedErr: true},
	}

	for _, tt := range tests {
		b, err := ParseToBytes(tt.input, tt.dataByteWidth)

		if tt.expectedErr {
			assert.True(t, err != nil)
		} else {
			assert.NoError(t, err, "input: "+tt.input)
		}

		assert.Equal(t, tt.expected, b)
	}
}

func TestNumberParse(t *testing.T) {
	tests := []struct {
		input       string
		expected    uint64
		expectedErr bool
	}{
		{input: "0", expected: 0},
		{input: "$ABCD", expected: 0xABCD},
		{input: "12345", expected: 12345},
		{input: "%01010101", expected: 85},
		{input: "01010101b", expected: 85},
		{input: "#%10001000", expected: 136},
		{input: "0xab", expected: 0xab},
		{input: "0ABhCDh", expectedErr: true},
		{input: "$0ABCDh", expectedErr: true},
		{input: "%01010101b", expectedErr: true},
		{input: "%%1", expectedErr: true},
		{input: "$0x1", expectedErr: true},
		{input: "0x12345678901234567890", expectedErr: true},
		{input: "%2", expectedErr: true},
	}

	for _, tt := range tests {
		i, err := Parse(tt.input)

		if tt.expectedErr {
			assert.True(t, err != nil)
		} else {
			assert.NoError(t, err, "input: "+tt.input)
		}

		assert.Equal(t, tt.expected, i)
	}
}
