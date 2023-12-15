package assembler

import "fmt"

// updateDataSizesStep updates the size information of data nodes.
func updateDataSizesStep(asm *Assembler) error {
	for _, seg := range asm.segmentsOrder {
		for _, node := range seg.nodes {
			dat, ok := node.(*data)
			if !ok {
				continue
			}

			if err := updateDataSize(dat); err != nil {
				return err
			}
		}
	}
	return nil
}

func updateDataSize(dat *data) error {
	// calculate complete size for normal data, fill data has the final size already set
	if dat.fill {
		if dat.width == 1 || dat.size.IsEvaluatedAtAddressAssign() {
			return nil
		}
		size, err := dat.size.IntValue()
		if err != nil {
			return fmt.Errorf("getting data node size: %w", err)
		}
		size *= int64(dat.width)
		dat.size.SetValue(size)
		return nil
	}

	size := 0

	for _, value := range dat.values {
		switch v := value.(type) {
		case int64, reference:
			size += dat.width

		case []byte:
			size += len(v)
		}
	}

	dat.size.SetValue(int64(size))
	return nil
}
