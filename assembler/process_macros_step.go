package assembler

import (
	"fmt"

	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/parser"
	"github.com/retroenv/assembler/parser/ast"
)

// processMacrosStep processes macro usages and replace them by the macro nodes.
func processMacrosStep(asm *Assembler) error {
	asm.currentScope = asm.fileScope

	for i, seg := range asm.segmentsOrder {
		segmentNodesResolved := make([]any, 0, len(seg.nodes))

		for _, node := range seg.nodes {
			switch n := node.(type) {
			case *ast.Identifier:
				nodes, err := resolveMacroUsage(asm, n)
				if err != nil {
					return fmt.Errorf("processing identifier '%s': %w", n.Name, err)
				}
				segmentNodesResolved = append(segmentNodesResolved, nodes...)

			case macro:
				_, ok := asm.macros[n.name]
				if ok {
					return fmt.Errorf("macro '%s' already exists", n.name)
				}
				asm.macros[n.name] = n

			case scopeChange:
				asm.currentScope = n.scope

			default:
				segmentNodesResolved = append(segmentNodesResolved, n)
			}
		}

		asm.segmentsOrder[i].nodes = segmentNodesResolved
	}

	return nil
}

func resolveMacroUsage(asm *Assembler, id *ast.Identifier) ([]any, error) {
	mac, ok := asm.macros[id.Name]
	if !ok {
		return nil, fmt.Errorf("macro '%s' not found", id.Name)
	}

	if len(mac.arguments) != len(id.Arguments) {
		return nil, fmt.Errorf("macro argument count %d does not match usage argument count %d",
			len(mac.arguments), len(id.Arguments))
	}

	// replace the macro placeholders with the passed values
	for i, tok := range mac.token {
		if tok.Type != token.Identifier {
			continue
		}

		argPos, ok := mac.arguments[tok.Value]
		if !ok {
			continue
		}

		arg := id.Arguments[argPos]

		// handle case for usage of #arg for a macro argument
		if i > 0 && mac.token[i-1].Type == token.Number && mac.token[i-1].Value == "#" {
			mac.token[i-1].Value = "#" + arg.Value
			mac.token[i-1].Type = arg.Type
			mac.token[i].Type = token.EOL
		} else {
			mac.token[i] = arg
		}
	}

	// convert the adjusted tokens to AST nodes
	par := parser.NewWithTokens(asm.cfg.Arch, mac.token)
	astNodes, err := par.TokensToAstNodes()
	if err != nil {
		return nil, fmt.Errorf("converting tokens to ast nodes: %w", err)
	}

	// process the AST nodes
	var result []any
	for _, node := range astNodes {
		nodes, err := parseASTNode(asm, node)
		if err != nil {
			return nil, err
		}

		result = append(result, nodes...)
	}

	return result, nil
}
