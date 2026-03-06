package parser

import "strings"

// Register types for special registers.
const (
	regSR  = 0xFF
	regCCR = 0xFE
	regUSP = 0xFD
	regPC  = 0xFC
)

type registerInfo struct {
	number  uint8
	isAddr  bool
	special bool // SR, CCR, USP, PC
}

var registers map[string]registerInfo

func init() {
	registers = make(map[string]registerInfo, 32)

	for i := uint8(0); i < 8; i++ {
		registers["D"+string(rune('0'+i))] = registerInfo{number: i}
	}

	for i := uint8(0); i < 7; i++ {
		registers["A"+string(rune('0'+i))] = registerInfo{number: i, isAddr: true}
	}
	registers["A7"] = registerInfo{number: 7, isAddr: true}
	registers["SP"] = registerInfo{number: 7, isAddr: true}

	registers["SR"] = registerInfo{number: regSR, special: true}
	registers["CCR"] = registerInfo{number: regCCR, special: true}
	registers["USP"] = registerInfo{number: regUSP, special: true, isAddr: true}
	registers["PC"] = registerInfo{number: regPC, special: true}
}

// lookupRegister returns register info for a name, or false if not a register.
func lookupRegister(name string) (registerInfo, bool) {
	info, ok := registers[strings.ToUpper(name)]
	return info, ok
}

// isDataRegister returns true if the name is D0-D7.
func isDataRegister(name string) bool {
	info, ok := lookupRegister(name)
	return ok && !info.isAddr && !info.special
}

// isAddrRegister returns true if the name is A0-A7/SP.
func isAddrRegister(name string) bool {
	info, ok := lookupRegister(name)
	return ok && info.isAddr && !info.special
}
