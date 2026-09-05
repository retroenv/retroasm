package ast

import (
	"reflect"
	"testing"
)

func BenchmarkEqual(b *testing.B) {
	inst := NewInstruction("lda", 1, NewNumber(1), nil)
	for name, pair := range map[string][2]Node{
		"instruction/shared":  {inst, inst},
		"instruction/copy":    {inst, inst.Copy()},
		"instruction/pointer": {&inst, &inst},
		"label":               {NewLabel("entry"), NewLabel("entry")},
		"identifier":          {NewIdentifier("value"), NewIdentifier("value")},
		"directive":           {NewData(DataType, 1), NewData(DataType, 1)},
	} {
		for comparison, equal := range map[string]func(Node, Node) bool{
			"typed": Equal, "reflection": equalReflection,
		} {
			b.Run(name+"/"+comparison, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					equal(pair[0], pair[1])
				}
			})
		}
	}
}

func equalReflection(left, right Node) bool { return reflect.DeepEqual(left, right) }
