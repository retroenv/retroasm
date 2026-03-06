package parser

import (
	"fmt"
	"strings"
)

// parseRegisterList parses a MOVEM register list like "D0-D3/A0-A2".
// Returns a 16-bit bitmask: bits 0-7 = D0-D7, bits 8-15 = A0-A7.
func parseRegisterList(s string) (uint16, error) {
	var mask uint16

	parts := strings.Split(s, "/")
	for _, part := range parts {
		rangeParts := strings.SplitN(part, "-", 2)
		if len(rangeParts) == 1 {
			bit, err := registerBit(rangeParts[0])
			if err != nil {
				return 0, err
			}
			mask |= 1 << bit
		} else {
			start, err := registerBit(rangeParts[0])
			if err != nil {
				return 0, err
			}
			end, err := registerBit(rangeParts[1])
			if err != nil {
				return 0, err
			}
			if end < start {
				return 0, fmt.Errorf("invalid register range %s", part)
			}
			for i := start; i <= end; i++ {
				mask |= 1 << i
			}
		}
	}

	return mask, nil
}

// registerBit returns the bit position (0-15) for a register name.
func registerBit(name string) (uint8, error) {
	info, ok := lookupRegister(strings.TrimSpace(name))
	if !ok || info.special {
		return 0, fmt.Errorf("invalid register in list: %s", name)
	}
	if info.isAddr {
		return 8 + info.number, nil
	}
	return info.number, nil
}
