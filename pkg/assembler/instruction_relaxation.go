package assembler

import (
	"context"
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
)

// assignAddressesStep retains one pass for fixed-size architectures. Targets
// with relaxation repeat layout after each growth pass. The number of nodes
// bounds the number of passes for targets with one growth per instruction.
func assignAddressesStep[T any](ctx context.Context, asm *Assembler[T]) error {
	relaxer, variableSize := asm.cfg.Arch.(arch.InstructionRelaxer)
	limit := 1
	if variableSize {
		for _, segment := range asm.segmentsOrder {
			limit += len(segment.nodes)
		}
	}

	for range limit {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("assigning addresses: %w", err)
		}
		if err := assignAddressPass(ctx, asm); err != nil {
			return err
		}
		if !variableSize {
			return nil
		}
		changed, err := relaxInstructions(asm, relaxer)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
	}

	return errors.New("instruction address relaxation did not converge")
}

func relaxInstructions[T any](asm *Assembler[T], relaxer arch.InstructionRelaxer) (bool, error) {
	assigner := addressAssign[T]{
		arch:         asm.cfg.Arch,
		currentScope: asm.fileScope,
	}
	changed := false
	for _, segment := range asm.segmentsOrder {
		for _, node := range segment.nodes {
			switch value := node.(type) {
			case scopeChange:
				assigner.currentScope = value.scope

			case *instruction:
				assigner.programCounter = value.Address()
				grown, err := relaxer.RelaxInstruction(&assigner, value)
				if err != nil {
					return false, fmt.Errorf("relaxing instruction %s: %w", value.Name(), err)
				}
				changed = changed || grown
			}
		}
	}

	return changed, nil
}
