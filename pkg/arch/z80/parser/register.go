package parser

import (
	"slices"
	"strings"

	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

var registerParamByName = map[string]cpuz80.RegisterParam{
	"a":  cpuz80.RegA,
	"af": cpuz80.RegAF,
	"b":  cpuz80.RegB,
	"bc": cpuz80.RegBC,
	"c":  cpuz80.RegC,
	"d":  cpuz80.RegD,
	"de": cpuz80.RegDE,
	"e":  cpuz80.RegE,
	"h":  cpuz80.RegH,
	"hl": cpuz80.RegHL,
	"i":  cpuz80.RegI,
	"ix": cpuz80.RegIX,
	"iy": cpuz80.RegIY,
	"l":  cpuz80.RegL,
	"r":  cpuz80.RegR,
	"sp": cpuz80.RegSP,
}

var conditionParamByName = map[string]cpuz80.RegisterParam{
	"c":  cpuz80.RegCondC,
	"m":  cpuz80.RegCondM,
	"nc": cpuz80.RegCondNC,
	"nz": cpuz80.RegCondNZ,
	"p":  cpuz80.RegCondP,
	"pe": cpuz80.RegCondPE,
	"po": cpuz80.RegCondPO,
	"z":  cpuz80.RegCondZ,
}

var registerParamsByNumber = map[uint64][]cpuz80.RegisterParam{
	0x00: {cpuz80.RegRst00, cpuz80.RegIM0},
	0x01: {cpuz80.RegIM1},
	0x02: {cpuz80.RegIM2},
	0x08: {cpuz80.RegRst08},
	0x10: {cpuz80.RegRst10},
	0x18: {cpuz80.RegRst18},
	0x20: {cpuz80.RegRst20},
	0x28: {cpuz80.RegRst28},
	0x30: {cpuz80.RegRst30},
	0x38: {cpuz80.RegRst38},
}

func registerCandidatesForIdentifier(value string) []cpuz80.RegisterParam {
	value = strings.ToLower(value)

	params := make([]cpuz80.RegisterParam, 0, 2)
	if registerParam, ok := registerParamByName[value]; ok {
		params = append(params, registerParam)
	}
	if conditionParam, ok := conditionParamByName[value]; ok && !containsRegisterParam(params, conditionParam) {
		params = append(params, conditionParam)
	}
	return params
}

func registerOnlyCandidate(value string) (cpuz80.RegisterParam, bool) {
	registerParam, ok := registerParamByName[strings.ToLower(value)]
	return registerParam, ok
}

func registerCandidatesForNumber(value uint64) []cpuz80.RegisterParam {
	candidates, ok := registerParamsByNumber[value]
	if !ok {
		return nil
	}

	result := make([]cpuz80.RegisterParam, len(candidates))
	copy(result, candidates)
	return result
}

func containsRegisterParam(params []cpuz80.RegisterParam, target cpuz80.RegisterParam) bool {
	return slices.Contains(params, target)
}
