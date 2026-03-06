package parser

import "strings"

// conditionCodes maps condition suffixes to their 4-bit encoding.
var conditionCodes = map[string]uint16{
	"T":  0,  // True (always)
	"F":  1,  // False (never)
	"HI": 2,  // High
	"LS": 3,  // Low or Same
	"CC": 4,  // Carry Clear
	"HS": 4,  // High or Same (alias for CC)
	"CS": 5,  // Carry Set
	"LO": 5,  // Low (alias for CS)
	"NE": 6,  // Not Equal
	"EQ": 7,  // Equal
	"VC": 8,  // Overflow Clear
	"VS": 9,  // Overflow Set
	"PL": 10, // Plus
	"MI": 11, // Minus
	"GE": 12, // Greater or Equal
	"LT": 13, // Less Than
	"GT": 14, // Greater Than
	"LE": 15, // Less or Equal
}

// ParseConditionCode extracts a condition code from a branch/set mnemonic.
// Returns the base instruction name, condition code, and whether a condition was found.
// Examples: "BEQ" -> ("Bcc", 7, true), "DBNE" -> ("DBcc", 6, true), "SHI" -> ("Scc", 2, true).
func ParseConditionCode(mnemonic string) (string, uint16, bool) {
	upper := strings.ToUpper(mnemonic)

	// Check for Bcc (branch conditional)
	if len(upper) >= 3 && upper[0] == 'B' {
		suffix := upper[1:]
		if code, ok := conditionCodes[suffix]; ok {
			return "Bcc", code, true
		}
	}

	// Check for DBcc (decrement and branch)
	if len(upper) >= 4 && upper[:2] == "DB" {
		suffix := upper[2:]
		if code, ok := conditionCodes[suffix]; ok {
			return "DBcc", code, true
		}
	}

	// Check for Scc (set conditional)
	if len(upper) >= 2 && upper[0] == 'S' {
		suffix := upper[1:]
		if code, ok := conditionCodes[suffix]; ok {
			return "Scc", code, true
		}
	}

	return mnemonic, 0, false
}
