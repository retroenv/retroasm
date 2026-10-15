package parser

import "strings"

// Register types for special registers.
const (
	RegSR  = 0xFF
	RegCCR = 0xFE
	regUSP = 0xFD
	regPC  = 0xFC
)

var registers map[string]registerInfo

// IsRegisterName reports whether a name identifies a CPU68000 register.
func IsRegisterName(name string) bool {
	_, ok := registers[strings.ToUpper(name)]
	return ok
}

func init() {
	registers = make(map[string]registerInfo, 32)

	for i := range uint8(8) {
		registers["D"+string(rune('0'+i))] = registerInfo{
			number: i,
		}
	}

	for i := range uint8(7) {
		registers["A"+string(rune('0'+i))] = registerInfo{
			number: i,
			isAddr: true,
		}
	}
	registers["A7"] = registerInfo{
		number: 7,
		isAddr: true,
	}
	registers["SP"] = registerInfo{
		number: 7,
		isAddr: true,
	}

	registers["SR"] = registerInfo{
		number:  RegSR,
		special: true,
	}
	registers["CCR"] = registerInfo{
		number:  RegCCR,
		special: true,
	}
	registers["USP"] = registerInfo{
		number:  regUSP,
		special: true,
		isAddr:  true,
	}
	registers["PC"] = registerInfo{
		number:  regPC,
		special: true,
	}
}

type registerInfo struct {
	number  uint8
	isAddr  bool
	special bool // SR, CCR, USP, PC
}

// lookupRegister returns register info for a name, or false if not a register.
func lookupRegister(name string) (registerInfo, bool) {
	info, ok := registers[strings.ToUpper(name)]
	return info, ok
}
