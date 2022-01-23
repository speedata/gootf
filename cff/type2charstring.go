package cff

import (
	"fmt"
	"log"
)

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

type type2state struct {
	stack         []int
	cHints        int
	hasWd         bool
	defaultWidthX int
	nominalWidthX int
	width         int
}

func (state *type2state) clearStack() {
	state.stack = state.stack[:0]
}

func (state *type2state) pop() int {
	var i int
	i, state.stack = state.stack[len(state.stack)-1], state.stack[:len(state.stack)-1]
	return i
}

func (state *type2state) popN(n int) {
	state.stack = state.stack[:len(state.stack)-n]
}

func (state *type2state) push(n int) {
	state.stack = append(state.stack, n)
}

func (state *type2state) clearEven() int {
	halfEvenStack := len(state.stack) / 2
	state.stack = state.stack[:len(state.stack)-halfEvenStack*2]
	return halfEvenStack
}

// checkWd returns true if the last argument has read as a width.
func (state *type2state) checkWd() bool {
	if state.hasWd {
		return false
	}
	if len(state.stack) == 0 {
		return false
	}
	num := state.stack[0]
	if num != state.defaultWidthX {
		state.stack = state.stack[1:]
		state.width = state.nominalWidthX + num
		state.hasWd = true
		return true
	}
	return false
}

// getSubrsIndex goes recursively into all subroutines called by the char string cs and
// sets the entries in the global maps usedGlobalSubrsMap and usedLocalSubrsMap to true
// if the subroutine is used.
func getSubrsIndex(nominalWidthX int, defaultWidthX int, globalSubrs [][]byte, localSubrs [][]byte, cs []byte, state *type2state) {
	if state == nil {
		state = &type2state{}
		state.stack = make([]int, 0, 48)
		state.nominalWidthX = nominalWidthX
		state.defaultWidthX = defaultWidthX
	}

	localBias := calculateBias(localSubrs)
	globalBias := calculateBias(globalSubrs)

	pos := -1
	for {
		pos++
		if len(cs) <= pos {
			break
		}
		b0 := cs[pos]
		if b0 == 1 {
			// hstem
			state.cHints += state.clearEven()
			state.checkWd()
		} else if b0 == 3 {
			// vstem
			state.cHints += state.clearEven()
			state.checkWd()
		} else if b0 == 4 {
			// vmoveto
			state.clearStack()
		} else if b0 == 5 {
			// rlineto
			state.clearStack()
		} else if b0 == 6 {
			// hlineto
			state.clearStack()
		} else if b0 == 7 {
			// vlineto
			state.clearStack()
		} else if b0 == 8 {
			// rrcurveto
			state.clearStack()
		} else if b0 == 10 {
			subrIdx := state.pop() + localBias

			if subrIdx < len(localSubrs) {
				getSubrsIndex(nominalWidthX, defaultWidthX, globalSubrs, localSubrs, localSubrs[subrIdx], state)
				usedLocalSubrsMap[subrIdx] = true
			} else {
				panic("subrIdx must be < len(localSubrs)")
			}
			state.checkWd()
		} else if b0 == 11 {
			// return
		} else if b0 == 12 {
			// escape
			b1 := cs[pos+1]
			pos++
			if b1 == 1 {

			}
		} else if b0 == 14 {
			// endchar
		} else if b0 == 18 {
			// hstemhm
			state.cHints += state.clearEven()
			state.checkWd()
		} else if b0 == 19 {
			// hintmask
			state.cHints += state.clearEven()
			// advance over then number several bytes: each hint has one bit, so
			// 1-8 hints use one byte, 9-16 use two bytes and so on
			bits := state.cHints / 8
			if state.cHints%8 != 0 {
				bits++
			}

			for i := 0; i < bits; i++ {
				pos++
			}
		} else if b0 == 20 {
			// cntrmask
			for i := 0; i <= state.cHints/8; i++ {
				pos++
			}
		} else if b0 == 21 {
			// rmoveto
			state.popN(2)
		} else if b0 == 22 {
			// hmoveto
			state.clearStack()
		} else if b0 == 23 {
			// vstemhm
			state.cHints += state.clearEven()
		} else if b0 == 24 {
			// rcurveline
			state.clearStack()
		} else if b0 == 25 {
			// rlinecurve
			state.clearStack()
		} else if b0 == 26 {
			// vvcurveto
			state.clearStack()
		} else if b0 == 27 {
			// hhcurveto
			state.clearStack()
		} else if b0 == 28 {
			// shortint
			log.Fatal("nyi: shortint")
		} else if b0 == 29 {
			subrIdx := state.pop() + globalBias

			if subrIdx < len(globalSubrs) {
				getSubrsIndex(nominalWidthX, defaultWidthX, globalSubrs, localSubrs, globalSubrs[subrIdx], state)
				usedGlobalSubrsMap[subrIdx] = true
			} else {
				panic("subrIdx must be < len(globalSubrs)")
			}
			state.checkWd()
		} else if b0 == 30 {
			// vhcurveto
			state.clearStack()
		} else if b0 == 31 {
			// hvcurveto
			state.clearStack()
		} else if b0 >= 32 && b0 <= 246 {
			var val int
			val = int(b0) - 139
			state.push(val)
		} else if b0 >= 247 && b0 <= 250 {
			b1 := cs[pos+1]
			pos++
			var val int
			val = (int(b0)-247)*256 + int(b1) + 108
			state.push(val)
		} else if b0 >= 251 && b0 <= 254 {
			b1 := cs[pos+1]
			pos++
			val := -(int(b0)-251)*256 - int(b1) - 108
			state.push(val)
		} else if b0 == 255 {
			pos += 4
			// ignore for now
			// state.clearStack()
		} else {
			fmt.Println("b", b0)
			// state.clearStack()
		}
	}
}
