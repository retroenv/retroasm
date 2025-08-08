// Package number provides comprehensive number parsing for retro computer assembly language.
//
// This package handles the various number formats commonly used in assembly programming
// for retro computers like NES, where different number representations are standard.
//
// # Supported Number Formats
//
// The parser supports all major number formats used in assembly:
//
// Decimal Numbers:
//   - Simple: 123, 255
//   - Immediate: #123, #255
//
// Hexadecimal Numbers:
//   - Dollar prefix: $FF, $ABCD
//   - C-style prefix: 0xFF, 0xABCD
//   - Suffix style: FFh, ABCDh (Note: suffix conversion to 0x prefix)
//   - Immediate: #$FF, #0xFF
//
// Binary Numbers:
//   - Percent prefix: %11110000, %10101010
//   - Suffix style: 11110000b, 10101010b
//   - Immediate: #%11110000
//
// # Data Width Conversion
//
// Numbers can be converted to byte arrays with specific widths for target hardware:
//   - 1 byte: 0-255 (uint8)
//   - 2 bytes: 0-65535 (uint16, little-endian)
//   - 4 bytes: 0-4294967295 (uint32, little-endian)
//   - 8 bytes: full uint64 range (little-endian)
//
// # Example Usage
//
//	// Parse various number formats
//	decimal, err := number.Parse("255")        // decimal
//	hex, err := number.Parse("$FF")           // hex with $ prefix
//	binary, err := number.Parse("%11111111")  // binary
//	immediate, err := number.Parse("#$FF")    // immediate hex
//
//	// Convert to byte arrays for target hardware
//	bytes1, err := number.ParseToBytes("$FF", 1)     // []byte{0xFF}
//	bytes2, err := number.ParseToBytes("$1234", 2)   // []byte{0x34, 0x12} (little-endian)
//	bytes4, err := number.ParseToBytes("$12345678", 4) // 4-byte little-endian
//
// # Error Handling
//
// The package uses structured errors with sentinel values for consistent error handling:
//   - ErrInvalidNumberBaseCombination: Multiple base prefixes (e.g., "$0xFF")
//   - ErrInvalidBinaryChar: Non-binary digits in binary numbers (e.g., "%12")
//   - ErrInvalidHexChar: Invalid hex characters
//   - ErrNumberExceedsWidth: Number too large for specified byte width
//   - ErrUnsupportedDataWidth: Invalid data width specified
//
// All errors preserve context and can be tested with errors.Is().
package number
