package cff

import (
	"fmt"
	"io"
	"math"
	"sort"
)

// Top DICT Data - see CFF spec 9 p. 14
func (f *Font) parseDict(dict []byte) {
	f.bluefuzz = 1
	f.blueshift = 7
	f.bluescale = 0.039625

	operands := make([]int, 0, 48)
	operandsf := make([]float64, 0, 48)
	pos := -1
	for {
		pos++
		if len(dict) <= pos {
			return
		}
		b0 := dict[pos]
		if b0 == 0 {
			// version
			f.version = SID(operands[0])
			operands = operands[:0]
		} else if b0 == 1 {
			// notice
			f.notice = SID(operands[0])
			operands = operands[:0]
		} else if b0 == 2 {
			// fullname
			f.fullname = SID(operands[0])
			operands = operands[:0]
		} else if b0 == 3 {
			f.familyname = SID(operands[0])
			operands = operands[:0]
		} else if b0 == 4 {
			// weight
			f.weight = SID(operands[0])
			operands = operands[:0]
		} else if b0 == 5 {
			// font bbox
			f.bbox = make([]int, 4)
			copy(f.bbox, operands)
			operands = operands[:0]
		} else if b0 == 6 {
			// Blue Values
			f.bluevalues = make([]int, len(operands))
			copy(f.bluevalues, operands)
			operands = operands[:0]
		} else if b0 == 7 {
			f.otherblues = make([]int, len(operands))
			copy(f.otherblues, operands)
			operands = operands[:0]
		} else if b0 == 8 {
			f.familyblues = make([]int, len(operands))
			copy(f.familyblues, operands)
			operands = operands[:0]
		} else if b0 == 9 {
			f.familyotherblues = make([]int, len(operands))
			copy(f.familyotherblues, operands)
			operands = operands[:0]
		} else if b0 == 10 {
			f.stdhw = operands[0]
			operands = operands[:0]
		} else if b0 == 11 {
			f.stdvw = operands[0]
			operands = operands[:0]
		} else if b0 == 12 {
			// two bytes
			pos++
			b1 := dict[pos]
			switch b1 {
			case 0:
				f.copyright = SID(operands[0])
			case 3:
				if len(operands) > 0 {
					f.underlinePosition = float64(operands[0])
				} else if len(operandsf) > 0 {
					f.underlinePosition = operandsf[0]
				}
			case 4:
				if len(operands) > 0 {
					f.underlineThickness = float64(operands[0])
				} else if len(operandsf) > 0 {
					f.underlineThickness = operandsf[0]
				}
			case 8:
				// StrokeWidth
			case 9:
				f.bluescale = operandsf[0]
				operands = operands[:0]
			case 10:
				f.blueshift = operands[0]
				operands = operands[:0]
			case 11:
				f.bluefuzz = operands[0]
				operands = operands[:0]
			case 12:
				f.stemsnaph = make([]int, len(operands))
				copy(f.stemsnaph, operands)
				operands = operands[:0]
			case 13:
				f.stemsnapv = make([]int, len(operands))
				copy(f.stemsnapv, operands)
				operands = operands[:0]
			case 19:
				f.initialRandomSeed = operands[0]
				operands = operands[:0]
			case 30:
				// ROS
				f.registry = SID(operands[0])
				f.ordering = SID(operands[1])
				f.supplement = operands[2]
				operands = operands[:2]
			case 34:
				// CID count
				f.cidcount = operands[0]
				operands = operands[:0]
			case 36:
				// FDArray
				f.fdarray = int64(operands[0])
				operands = operands[:0]
			case 37:
				// FDSelect
				f.fdselect = int64(operands[0])
				operands = operands[:0]
			case 38:
				// fontname
				f.name = SID(operands[0])
				operands = operands[:0]
			default:
				panic(fmt.Sprintf("not implemented ESC %d", b1))
			}
			operands = operands[:0]
			operandsf = operandsf[:0]
		} else if b0 == 13 {
			// unique id
			f.uniqueid = operands[0]
			operands = operands[:0]
		} else if b0 == 15 {
			// charset
			f.charsetOffset = int64(operands[0])
			operands = operands[:0]
		} else if b0 == 16 {
			f.encodingOffset = operands[0]
			operands = operands[:0]
		} else if b0 == 17 {
			// charstrings (type 2 instructions)
			f.charstringsOffset = int64(operands[0])
			operands = operands[:0]
		} else if b0 == 18 {
			f.privatedictsize = operands[0]
			f.privatedictoffset = int64(operands[1])
			operands = operands[:1]
		} else if b0 == 19 {
			f.subrsOffset = operands[0]
			operands = operands[:0]
		} else if b0 == 20 {
			f.defaultWidthX = operands[0]
			operands = operands[:0]
		} else if b0 == 21 {
			f.nominalWidthX = operands[0]
			operands = operands[:0]
		} else if b0 == 28 {
			b1 := dict[pos+1]
			b2 := dict[pos+2]
			pos += 2
			val := int(b1)<<8 | int(b2)
			operands = append(operands, val)
		} else if b0 == 29 {
			b1 := dict[pos+1]
			b2 := dict[pos+2]
			b3 := dict[pos+3]
			b4 := dict[pos+4]
			pos += 4
			val := int(b1)<<24 | int(b2)<<16 | int(b3)<<8 | int(b4)
			operands = append(operands, val)
		} else if b0 == 30 {
			// float
			valbefore := 0
			valafter := 0
			digitsafter := 0
			mode := "before"
			shift := 1
		parsefloat:
			for {
				b1 := dict[pos+1]
				pos++
				n1, n2 := b1>>4, b1&0xf
				nibble := n1
				firstnibble := true
				for {
					if nibble == 0xf {
						break parsefloat
					} else if nibble >= 0 && nibble <= 9 {
						if mode == "before" {
							valbefore = 10*valbefore + int(nibble)
						} else if mode == "after" {
							valafter = 10*valafter + int(nibble)
							digitsafter++
						} else if mode == "E-" {
							shift = int(nibble) * -1
						} else if mode == "E" {
							shift = int(nibble)
						}
					} else if nibble == 0xa {
						mode = "after"
					} else if nibble == 0xb {
						mode = "E"
					} else if nibble == 0xc {
						mode = "E-"
					} else if nibble == 0xe {
						valbefore = valbefore * -1
					}
					if firstnibble {
						nibble = n2
						firstnibble = false
					} else {
						break
					}
				}
			}
			div := 1
			for i := 0; i < digitsafter; i++ {
				div *= 10
			}
			var flt = float64(valbefore)
			flt += (float64(valafter) / float64(div))
			flt = math.Pow(flt, float64(shift))
			operandsf = append(operandsf, flt)
		} else if b0 >= 32 && b0 <= 246 {
			val := int(b0) - 139
			operands = append(operands, val)
		} else if b0 >= 247 && b0 <= 250 {
			b1 := dict[pos+1]
			pos++
			val := (int(b0)-247)*256 + int(b1) + 108
			operands = append(operands, val)
		} else if b0 >= 251 && b0 <= 254 {
			b1 := dict[pos+1]
			pos++
			val := -(int(b0)-251)*256 - int(b1) - 108
			operands = append(operands, val)
		} else {
			fmt.Println("b0", b0)
			panic("not implemented yet")
		}
	}
}

func (f *Font) readCharStringsIndex(r io.ReadSeeker) error {
	if _, err := r.Seek(f.charstringsOffset, io.SeekStart); err != nil {
		return err
	}
	data := cffReadIndexData(r, "CharStrings")

	f.CharStrings = data
	return nil
}

func (f *Font) readSubrIndex(r io.ReadSeeker) error {
	if f.subrsOffset == 0 {
		return nil
	}
	if _, err := r.Seek(f.privatedictoffset+int64(f.subrsOffset), io.SeekStart); err != nil {
		return err
	}
	data := cffReadIndexData(r, "Local Subrs")
	f.subrsIndex = data
	return nil
}

func (f *Font) readEncoding(r io.ReadSeeker) error {
	var err error
	f.encoding = make(map[int]int)

	r.Seek(int64(f.encodingOffset), io.SeekStart)
	read(r, &f.encodingFormat)
	switch f.encodingFormat {
	case 0:
		var c uint8
		read(r, &c)
		var enc uint8
		// is this correct???
		for i := 0; i < int(c); i++ {
			read(r, &enc)
			f.encoding[i+1] = int(enc)
		}
	case 1:
		var nRanges uint8
		read(r, &nRanges)
		for i := 0; i < int(nRanges); i++ {
			var first uint8
			var nLeft uint8
			if err = read(r, first); err != nil {
				return err
			}
			if err = read(r, nLeft); err != nil {
				return err
			}
			// we don't need the encoding, so we ignore it
		}
	default:
		panic(fmt.Sprintf("not implemented yet: encoding format %d", f.encodingFormat))
	}
	return nil
}

// cffReadCharset reads the glyph names of the font
func (f *Font) readCharset(r io.ReadSeeker) error {
	if _, err := r.Seek(f.charsetOffset, io.SeekStart); err != nil {
		return err
	}
	numGlyphs := len(f.CharStrings)
	if numGlyphs == 0 {
		return fmt.Errorf("char strings table needs to be parsed before charset")
	}

	f.charset = make([]SID, numGlyphs)

	read(r, &f.charsetFormat)
	switch f.charsetFormat {
	case 0:
		if f.IsCIDFont() && false {
			panic("niy")
		} else {
			var sid uint16
			for i := 1; i < numGlyphs; i++ {
				read(r, &sid)
				f.charset[i] = SID(sid)
			}
		}
	case 1:
		// .notdef is always 0 and not in the charset
		glyphsleft := numGlyphs - 1

		var sid uint16
		var nleft byte
		c := 1
		for {
			glyphsleft--
			read(r, &sid)
			read(r, &nleft)
			glyphsleft = glyphsleft - int(nleft)
			for i := 0; i <= int(nleft); i++ {
				f.charset[c] = SID(int(sid) + i)
				c++
			}
			if glyphsleft <= 0 {
				break
			}
		}
	default:
		panic(fmt.Sprintf("not implemented: charset format %d", f.charsetFormat))
	}
	return nil
}

func (f *Font) readPrivateDict(r io.ReadSeeker) error {
	if _, err := r.Seek(f.privatedictoffset, io.SeekStart); err != nil {
		return err
	}
	data := make([]byte, f.privatedictsize)
	read(r, &data)
	f.privatedict = data
	f.parseDict(data)
	return nil
}

// GetRawIndexData returns a byte slice of the index
func (f *Font) GetRawIndexData(r io.ReadSeeker, index mainIndex) ([]byte, error) {
	var indexStart int64
	var err error

	switch index {
	case CharStringsIndex:
		indexStart, err = r.Seek(f.charstringsOffset, io.SeekStart)
		if err != nil {
			return nil, err
		}
		err = f.readCharStringsIndex(r)

	case CharSet:
		indexStart, err = r.Seek(f.charsetOffset, io.SeekStart)
		if err != nil {
			return nil, err
		}
		err = f.readCharset(r)
	case Encoding:
		indexStart, err = r.Seek(int64(f.encodingOffset), io.SeekStart)
		if err != nil {
			return nil, err
		}
		err = f.readEncoding(r)
	case PrivateDict:
		indexStart, err = r.Seek(f.privatedictoffset, io.SeekStart)
		if err != nil {
			return nil, err
		}
		err = f.readPrivateDict(r)
	case LocalSubrsIndex:
		if f.subrsOffset == 0 {
			return nil, nil
		}
		indexStart, err = r.Seek(f.privatedictoffset+int64(f.subrsOffset), io.SeekStart)
		if err != nil {
			return nil, err
		}
		err = f.readSubrIndex(r)

	default:
		panic(fmt.Sprintf("unknown index %d", index))
	}
	if err != nil {
		return nil, err
	}

	cur, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = r.Seek(indexStart, io.SeekStart)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, cur-indexStart)
	_, err = r.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// parseIndex parses the index starting at the current r position
func (f *Font) parseIndex(r io.ReadSeeker, index mainIndex) error {
	var err error
	switch index {
	case CharStringsIndex:
		err = f.readCharStringsIndex(r)
	case PrivateDict:
		err = f.readPrivateDict(r)
	case CharSet:
		err = f.readCharset(r)
	case LocalSubrsIndex:
		err = f.readSubrIndex(r)
	case Encoding:
		err = f.readEncoding(r)
	default:
		panic(fmt.Sprintf("unknown index %d", index))
	}
	if err != nil {
		return err
	}
	return nil
}

// IsCIDFont returns true if the character encoding is based on CID instead of SID
func (f *Font) IsCIDFont() bool {
	return f.fdselect != 0
}

// Subset changes the font so that only the given code points remain in the font. Subset must only be called once.
func (f *Font) Subset(globalSubr [][]byte, codepoints []int) {
	sort.Ints(codepoints)
	cpIdx := 0
	charstringsIdx := 0
	for {
		cp := codepoints[cpIdx]
		for j := 1; j+charstringsIdx < cp; j++ {
			f.CharStrings[j+charstringsIdx] = []byte{0xe}
			f.charset[j+charstringsIdx] = 0
		}
		cpIdx++
		charstringsIdx = cp
		if cpIdx >= len(codepoints) {
			break
		}
	}

	lastcp := codepoints[len(codepoints)-1]
	f.CharStrings = f.CharStrings[:lastcp+1]
	f.charset = f.charset[:lastcp+1]

	usedGlobalSubrsMap = make(map[int]bool)
	usedLocalSubrsMap = make(map[int]bool)

	for _, cp := range codepoints {
		cs := f.CharStrings[cp]
		getSubrsIndex(globalSubr, f.subrsIndex, cs)
	}

	clearSubr(globalSubr, usedGlobalSubrsMap)
	clearSubr(f.subrsIndex, usedLocalSubrsMap)
}

func clearSubr(subr [][]byte, usedSubrs map[int]bool) {
	if len(usedSubrs) == 0 {
		return
	}
	usedSubrSlice := []int{}

	for k := range usedSubrs {
		usedSubrSlice = append(usedSubrSlice, k)
	}

	sort.Ints(usedSubrSlice)

	subrIdx := 0
	i := 0
	for {
		subri := usedSubrSlice[subrIdx]
		for j := 0; j+i < subri; j++ {
			subr[j+i] = []byte{}
		}
		i = subri + 1
		subrIdx++

		if subrIdx >= len(usedSubrSlice) {
			break
		}
	}
	lastidx := usedSubrSlice[len(usedSubrSlice)-1]
	subr = subr[:lastidx+1]

}
