package assembler

import (
	"github.com/retroenv/retroasm/assembler/config"
	"github.com/retroenv/retroasm/parser/ast"
)

type segment struct {
	config *config.Segment
	nodes  []ast.Node
}

func (seg *segment) addNode(node ast.Node) {
	seg.nodes = append(seg.nodes, node)
}
