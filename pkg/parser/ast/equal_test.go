package ast

import (
	"math"
	"reflect"
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/assert"
)

type equalityExtension struct {
	*node

	Payload any
	Next    *equalityExtension
}

func (ext equalityExtension) Copy() Node { return ext }

func TestEqualMatchesReflection(t *testing.T) {
	t.Parallel()

	cases := equalityCases()
	for left, before := range cases {
		for right, after := range cases {
			assert.Equal(t, reflect.DeepEqual(before, after), Equal(before, after),
				"cases %d (%T) and %d (%T)", left, before, right, after)
		}
	}
}

func TestEqualCycles(t *testing.T) {
	t.Parallel()

	left := &Instruction{Name: "cycle"}
	left.Argument = left
	right := &Instruction{Name: "cycle"}
	right.Argument = right
	unequal := &Instruction{Name: "other"}
	unequal.Argument = unequal
	extension := &equalityExtension{Payload: math.NaN()}
	extension.Next = extension
	other := &equalityExtension{Payload: math.NaN()}
	other.Next = other
	cases := []Node{left, right, unequal, extension, other,
		Instruction{Argument: RegisterValue{Value: left}},
		Instruction{Argument: RegisterValue{Value: right}},
	}
	for _, before := range cases {
		for _, after := range cases {
			assert.Equal(t, reflect.DeepEqual(before, after), Equal(before, after))
		}
	}
}

func TestEqualFieldInventory(t *testing.T) {
	t.Parallel()

	// A new field needs an equality rule and a parity case.
	for typ, count := range map[reflect.Type]int{
		reflect.TypeFor[node](): 2, reflect.TypeFor[Comment](): 2,
		reflect.TypeFor[Instruction](): 6, reflect.TypeFor[OpcodeID](): 2,
		reflect.TypeFor[Number](): 2, reflect.TypeFor[Label](): 2,
		reflect.TypeFor[Identifier](): 3, reflect.TypeFor[Modifier](): 3,
		reflect.TypeFor[Operator](): 2, reflect.TypeFor[token.Token](): 3,
		reflect.TypeFor[token.Position](): 2,
	} {
		assert.Equal(t, count, typ.NumField(), "%s", typ)
	}
}

func TestEqualLeafAllocations(t *testing.T) {
	for _, before := range []Node{
		NewNumber(1), NewLabel("entry"), NewIdentifier("value"),
		NewInstruction("lda", 1, NewNumber(1), nil),
	} {
		after := before.Copy()
		assert.True(t, Equal(before, after))
		assert.Equal(t, float64(0), testing.AllocsPerRun(20, func() { Equal(before, after) }))
	}
}

func equalityCases() []Node {
	seeds := []Node{
		Instruction{}, Number{}, Label{}, Identifier{}, Operator{}, &Comment{},
		Alias{}, Bank{}, Base{}, Configuration{}, Data{}, If{}, Ifdef{}, Ifndef{},
		Else{}, ElseIf{}, Endif{}, Enum{}, EnumEnd{}, Error{}, Expression{}, Function{},
		FunctionEnd{}, Include{}, InstructionArgument{}, InstructionArguments{}, Macro{},
		OffsetCounter{}, RegisterValue{}, RegisterRegisterValue{}, Rept{}, Endr{}, Scope{},
		ScopeEnd{}, Segment{}, Variable{}, equalityExtension{Payload: []int{1, 2}},
	}
	cases := []Node{nil}
	for _, seed := range append(seeds, equalityFieldCases()...) {
		cases = append(cases, seed)
		typ := reflect.TypeOf(seed)
		if typ.Kind() == reflect.Pointer {
			continue
		}
		ptr := reflect.New(typ)
		ptr.Elem().Set(reflect.ValueOf(seed))
		cases = append(cases, ptr.Interface().(Node), reflect.Zero(ptr.Type()).Interface().(Node))
	}
	var nilComment *Comment
	return append(cases, nilComment, &Comment{Message: "comment"})
}

func equalityFieldCases() []Node {
	base := &node{}
	comment := &node{comment: Comment{Message: "comment"}}
	return []Node{
		Number{node: base}, Number{node: &node{}}, Number{node: comment}, Number{Value: 1},
		Label{node: base}, Label{node: &node{}}, Label{node: comment}, Label{Name: "name"},
		Identifier{node: base}, Identifier{node: &node{}}, Identifier{node: comment},
		Identifier{Name: "name"}, Identifier{Arguments: []token.Token{}},
		Identifier{Arguments: []token.Token{{}}},
		Identifier{Arguments: []token.Token{{Type: token.Number}}},
		Identifier{Arguments: []token.Token{{Value: "1"}}},
		Identifier{Arguments: []token.Token{{Position: token.Position{Line: 1}}}},
		Identifier{Arguments: []token.Token{{Position: token.Position{Column: 1}}}},
		Operator{node: base}, Operator{node: comment}, Operator{Operator: "+"},
		Instruction{node: base}, Instruction{node: &node{}}, Instruction{node: comment},
		Instruction{Name: "lda"}, Instruction{Addressing: 1},
		Instruction{OpcodeID: OpcodeID{Architecture: arch.CPU6502}}, Instruction{OpcodeID: OpcodeID{Value: 1}},
		Instruction{Argument: Number{}}, Instruction{Argument: Number{Value: 1}},
		Instruction{Argument: NewNumber(1)}, Instruction{Argument: NewNumber(1)},
		Instruction{Argument: Label{Name: "name"}}, Instruction{Argument: Identifier{Name: "name"}},
		Instruction{Argument: RegisterValue{Register: 1, Value: Number{Value: 1}}},
		Instruction{Argument: RegisterValue{Register: 2, Value: Number{Value: 1}}},
		Instruction{Modifier: []Modifier{}}, Instruction{Modifier: []Modifier{{}}},
		Instruction{Modifier: []Modifier{{node: *comment}}},
		Instruction{Modifier: []Modifier{{Value: "1"}}},
		Instruction{Modifier: []Modifier{{Operator: Operator{node: base}}}},
		Instruction{Modifier: []Modifier{{Operator: Operator{node: &node{}}}}},
		Instruction{Modifier: []Modifier{{Operator: Operator{node: comment}}}},
		Instruction{Modifier: []Modifier{{Operator: Operator{Operator: "+"}}}},
		Instruction{Modifier: []Modifier{{Value: "1"}, {}}},
		Instruction{Modifier: []Modifier{{}, {Value: "1"}}},
	}
}

func TestEqualIgnoresEntryHandles(t *testing.T) {
	for _, source := range []Node{
		NewInstruction("lda", 0, NewNumber(1), nil), NewNumber(1), NewLabel("entry"),
		NewIdentifier("symbol"), NewData(DataType, 1), &Comment{Message: "source"},
		RegisterValue{node: &node{}, Register: 1, Value: NewNumber(2)},
		NewAlias("alias"),
	} {
		left, right := source.Copy(), source.Copy()
		carrier := left.(interface{ setEntryHandle(*entryHandle) })
		carrier.setEntryHandle(&entryHandle{})
		assert.True(t, Equal(left, right), "%T", source)
		assert.True(t, Equal(right, left), "%T", source)
		assert.True(t, Equal(left, left.Copy()), "%T", source)
		right.SetComment("changed")
		assert.False(t, Equal(left, right), "%T", source)
	}
	left := &equalityExtension{node: &node{handle: &entryHandle{}}, Payload: []int{1}}
	left.Next = left
	right := &equalityExtension{node: &node{}, Payload: []int{1}}
	right.Next = right
	assert.True(t, Equal(left, right))
	right.Payload = []int{2}
	assert.False(t, Equal(left, right))
}

func TestEqualCompositeReflectionParity(t *testing.T) {
	cycle := make(map[string]any)
	cycle["self"] = cycle
	otherCycle := make(map[string]any)
	otherCycle["self"] = otherCycle
	channel := make(chan int)
	values := []any{
		nil, true, false, int64(1), uint64(1), "value", complex(1, 2), math.NaN(),
		[]int(nil), []int{}, []int{1}, []int{1}, [1]int{1}, [1]int{2},
		map[string]int(nil), map[string]int{}, map[string]int{"key": 1}, map[string]int{"other": 1},
		cycle, otherCycle, channel, make(chan int), (chan int)(nil), (func())(nil),
		func() {}, struct{ private any }{private: []int{1}},
	}
	for _, left := range values {
		for _, right := range values {
			before := &equalityExtension{Payload: left}
			after := &equalityExtension{Payload: right}
			assert.Equal(t, reflect.DeepEqual(before, after), Equal(before, after), "%T and %T", left, right)
		}
	}
}
