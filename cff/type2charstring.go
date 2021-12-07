package cff

func calculateBias(subrs [][]byte) int {
	if len(subrs) < 1240 {
		return 107
	}
	if len(subrs) < 33900 {
		return 1131
	}
	return 32768
}

var (
	usedGlobalSubrsMap map[int]bool
	usedLocalSubrsMap  map[int]bool
)

// getSubrsIndex goes recursively into all subroutines called by the char string cs and
// sets the entries in the global maps usedGlobalSubrsMap and usedLocalSubrsMap to true
// if the subroutine is used.
func getSubrsIndex(globalSubrs [][]byte, localSubrs [][]byte, cs []byte) {
	operands := make([]int, 0, 48)
	localBias := calculateBias(localSubrs)
	globalBias := calculateBias(globalSubrs)

	pos := -1
	for {
		pos++
		if len(cs) <= pos {
			break
		}
		b0 := cs[pos]
		if b0 == 10 {
			if subrIdx := operands[len(operands)-1] + localBias; subrIdx < len(localSubrs) {
				if _, ok := usedLocalSubrsMap[subrIdx]; !ok {
					getSubrsIndex(globalSubrs, localSubrs, localSubrs[subrIdx])
					usedLocalSubrsMap[subrIdx] = true
				}
			}
			operands = operands[:0]
		} else if b0 == 29 {
			if subrIdx := operands[len(operands)-1] + globalBias; subrIdx < len(globalSubrs) {
				if _, ok := usedGlobalSubrsMap[subrIdx]; !ok {
					getSubrsIndex(globalSubrs, localSubrs, globalSubrs[subrIdx])
					usedGlobalSubrsMap[subrIdx] = true
				}
			}
			operands = operands[:0]
		} else if b0 >= 32 && b0 <= 246 {
			var val int
			val = int(b0) - 139
			operands = append(operands, val)
		} else if b0 >= 247 && b0 <= 250 {
			b1 := cs[pos+1]
			pos++
			var val int
			val = (int(b0)-247)*256 + int(b1) + 108
			operands = append(operands, val)
		} else if b0 >= 251 && b0 <= 254 {
			b1 := cs[pos+1]
			pos++
			val := -(int(b0)-251)*256 - int(b1) - 108
			operands = append(operands, val)
		} else if b0 == 255 {
			pos += 4
			// ignore for now
			operands = operands[:0]
		} else {
			operands = operands[:0]
		}
	}
}
