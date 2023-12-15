package assembler

import "github.com/retroenv/assembler/assembler/config"

type segment struct {
	config *config.Segment
	nodes  []any
}

func (seg *segment) addNode(node any) {
	seg.nodes = append(seg.nodes, node)
}
