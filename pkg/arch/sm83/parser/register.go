// Package parser implements SM83-specific instruction parsing and variant resolution.
package parser

import (
	"strings"

	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
)

var registerParamByName = map[string]cpusm83.RegisterParam{
	"a":  cpusm83.RegA,
	"af": cpusm83.RegAF,
	"b":  cpusm83.RegB,
	"bc": cpusm83.RegBC,
	"c":  cpusm83.RegC,
	"d":  cpusm83.RegD,
	"de": cpusm83.RegDE,
	"e":  cpusm83.RegE,
	"h":  cpusm83.RegH,
	"hl": cpusm83.RegHL,
	"l":  cpusm83.RegL,
	"sp": cpusm83.RegSP,
}

var conditionParamByName = map[string]cpusm83.RegisterParam{
	"nc": cpusm83.RegCondNC,
	"nz": cpusm83.RegCondNZ,
	"z":  cpusm83.RegCondZ,
	// NOTE: "c" is NOT here — it's ambiguous with register C.
}

var indirectRegisterParamsByName = map[string]cpusm83.RegisterParam{
	"bc": cpusm83.RegBCIndirect,
	"de": cpusm83.RegDEIndirect,
	"hl": cpusm83.RegHLIndirect,
}

var rstVectorByValue = map[uint64]cpusm83.RegisterParam{
	0x00: cpusm83.RegRst00,
	0x08: cpusm83.RegRst08,
	0x10: cpusm83.RegRst10,
	0x18: cpusm83.RegRst18,
	0x20: cpusm83.RegRst20,
	0x28: cpusm83.RegRst28,
	0x30: cpusm83.RegRst30,
	0x38: cpusm83.RegRst38,
}

func lookupRegister(name string) (cpusm83.RegisterParam, bool) {
	param, ok := registerParamByName[strings.ToLower(name)]
	return param, ok
}

func lookupCondition(name string) (cpusm83.RegisterParam, bool) {
	param, ok := conditionParamByName[strings.ToLower(name)]
	return param, ok
}

func lookupIndirectRegister(name string) (cpusm83.RegisterParam, bool) {
	param, ok := indirectRegisterParamsByName[strings.ToLower(name)]
	return param, ok
}

func lookupRstVector(value uint64) (cpusm83.RegisterParam, bool) {
	param, ok := rstVectorByValue[value]
	return param, ok
}

func isCondition(param cpusm83.RegisterParam) bool {
	return param == cpusm83.RegCondNZ || param == cpusm83.RegCondZ ||
		param == cpusm83.RegCondNC || param == cpusm83.RegCondC
}
