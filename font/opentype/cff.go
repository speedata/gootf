package opentype

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
)

var (
	removeTrailingZeros *regexp.Regexp
)

func init() {
	removeTrailingZeros = regexp.MustCompile("\\.?0*(e[+-]?)0*")
}

func parseDelta(delta []int) []int {
	ret := make([]int, len(delta))
	prev := 0
	for i := 0; i < len(delta); i++ {
		val := delta[i] + prev
		ret[i] = val
		prev = val
	}
	return ret
}

func (tt *Font) cffReadNameIndex() {
	for _, entry := range tt.cffReadIndex() {
		tt.CFF.fontnames = append(tt.CFF.fontnames, string(entry))
	}
}

func (tt *Font) cffWriteNameIndex(w io.Writer) int {
	var data [][]byte
	data = make([][]byte, 0)
	for _, str := range tt.CFF.fontnames {
		data = append(data, []byte(str))
	}
	return tt.cffWriteIndex(w, data)
}

// cffReadIndex reads a number of slices with data.
func (tt *Font) cffReadIndex() [][]byte {
	var count uint16
	tt.read(&count)
	if count == 0 {
		return [][]byte{}
	}
	var offsetSize uint8
	tt.read(&offsetSize)
	offsets := make([]int, count+1)
	for i := 0; i < int(count+1); i++ {
		offsets[i] = int(tt.readOffset(offsetSize))
	}
	data := make([][]byte, count)
	for i := 0; i < int(count); i++ {
		data[i] = tt.readBytes(offsets[i+1] - offsets[i])
	}
	return data
}

// cffWriteIndex writes the data slices to the writer w in CFF index format (cf CFF spec 5 INDEX Data p. 12).
// It returns the total number of bytes written to the writer.
func (tt *Font) cffWriteIndex(w io.Writer, data [][]byte) int {
	count := uint16(len(data))
	tt.write(w, count)
	indexLen := 2
	if count == 0 {
		return indexLen
	}
	// TODO: offset size should depend on the actual offset values
	offsetSize := uint8(1)
	tt.write(w, offsetSize)
	tt.write(w, uint8(1))
	indexLen += 2
	for c, i := 0, 0; i < len(data); i++ {
		c += len(data[i])
		tt.write(w, uint8(c)+1)
	}
	indexLen += len(data)
	for _, b := range data {
		tt.write(w, b)
		indexLen += len(b)
	}
	return indexLen
}

func (tt *Font) readBytes(n int) []byte {
	b := make([]byte, n)
	l, err := tt.r.Read(b)
	if err != nil {
		panic(err)
	}
	if l != n {
		panic(errors.New("Not enough bytes read"))
	}
	return b
}

// readOffset returns the offset value used in the index data.
// It depends on offset size which can be one to four bytes.
func (tt *Font) readOffset(offsetsize uint8) uint32 {
	switch offsetsize {
	case 1:
		var offset uint8
		tt.read(&offset)
		return uint32(offset)
	case 2:
		var offset uint16
		tt.read(&offset)
		return uint32(offset)
	case 3:
		return tt.get3uint32()
	case 4:
		var offset uint32
		tt.read(&offset)
		return offset
	default:
		panic(fmt.Sprintf("not implemented offset size %d", offsetsize))
	}
}

// One dictionary for each subfont.
func (tt *Font) cffWriteDictIndex(w io.Writer) int {
	var data [][]byte
	for _, fnt := range tt.CFF.font {
		data = append(data, tt.cffEncodeDict(fnt))
	}
	return tt.cffWriteIndex(w, data)
}

// cffReadDictIndex creates the tt.CFF.font objects and fills the data for each font found in the dict index.
func (tt *Font) cffReadDictIndex() {
	for i, entry := range tt.cffReadIndex() {
		fnt := &CFFFont{
			global:             &tt.CFF,
			dict:               entry,
			name:               tt.CFF.fontnames[i],
			underlineThickness: 50,
			underlinePosition:  -100,
		}
		// we cannot parse the dict now, since we need the strings first
		tt.CFF.font = append(tt.CFF.font, fnt)
	}
}

// cffReadStringIndex fills the strings and stringToInt slices
func (tt *Font) cffReadStringIndex() {
	for _, entry := range tt.cffReadIndex() {
		str := string(entry)
		tt.CFF.strings = append(tt.CFF.strings, str)
		tt.CFF.stringToInt[str] = len(tt.CFF.strings) - 1
	}
}

// cffWriteStringIndex writes all (non-predefined) strings to the writer w.
// It returns the total number of bytes written to w.
func (tt *Font) cffWriteStringIndex(w io.Writer) int {
	var data [][]byte
	// only write the not-predefined strings
	for _, str := range tt.CFF.strings[len(predefinedStrings):] {
		data = append(data, []byte(str))
	}
	return tt.cffWriteIndex(w, data)
}

// cffReadCharset reads the glyph names of the font
func (tt *Font) cffReadCharset(v *CFFFont) {
	tt.r.Seek(v.charsetOffset, io.SeekStart)
	v.charset = make([]int, v.nglyphs)
	var format uint8
	tt.read(&format)
	switch format {
	case 0:
		if v.IsCIDFont() {
			panic("niy")
		} else {
			var sid uint16
			for i := 1; i < v.nglyphs; i++ {
				tt.read(&sid)
				v.charset[i] = int(sid)
			}
		}
	case 1:
		// .notdef is always 0 and not in the charset
		glyphsleft := v.nglyphs - 1
		var sid uint16
		var nleft byte
		c := 1
		for {
			glyphsleft--
			tt.read(&sid)
			tt.read(&nleft)
			glyphsleft = glyphsleft - int(nleft)
			for i := 0; i <= int(nleft); i++ {
				v.charset[c] = int(sid) + i
				c++
			}
			if glyphsleft <= 0 {
				break
			}
		}
	default:
		panic(fmt.Sprintf("not implemented: charset format %d", format))
	}
}

// cffWriteCharset writes the glyph names of the font to the writer w. It returns the number of bytes written to w.
func (tt *Font) cffWriteCharset(w io.Writer, v *CFFFont) int {
	format := uint8(0)
	tt.write(w, format)
	var sid uint16
	for i := 1; i < v.nglyphs; i++ {
		sid = uint16(v.charset[i])
		tt.write(w, sid)
	}
	return 1 + (v.nglyphs-1)*2
}

func (tt *Font) cffReadPrivateDict(v *CFFFont) {
	tt.r.Seek(v.privatedictoffset, io.SeekStart)
	data := make([]byte, v.privatedictsize)
	tt.read(&data)
	v.privatedict = data
}

// cffWritePrivateDict write the private dictionary of a CFFFont. It returns the number of bytes written to w.
func (tt *Font) cffWritePrivateDict(w io.Writer, v *CFFFont) int {
	tt.write(w, v.privatedict)
	return len(v.privatedict)
}

// Global subroutine Index
func (tt *Font) cffReadGlobalSubrIndex() {
	for _, entry := range tt.cffReadIndex() {
		tt.CFF.globalSubr = append(tt.CFF.globalSubr, entry)
	}
}

func (tt *Font) cffWriteGlobalSubrIndex(w io.Writer) int {
	return tt.cffWriteIndex(w, tt.CFF.globalSubr)
}

// The code for the glyphs
func (tt *Font) cffReadCharstring(v *CFFFont) {
	tt.r.Seek(v.charstringsOffset, io.SeekStart)
	for _, entry := range tt.cffReadIndex() {
		v.nglyphs++
		v.charString = append(v.charString, entry)
	}
}

func (tt *Font) cffReadEncoding(v *CFFFont) {
	if v.encodingOffset == 0 {
		return
	}
	v.encoding = make(map[int]int)
	tt.r.Seek(int64(v.encodingOffset), io.SeekStart)
	var version uint8
	tt.read(&version)
	switch version {
	case 0:
		var c uint8
		tt.read(&c)
		var enc uint8
		// is this correct???
		for i := 0; i < int(c); i++ {
			tt.read(&enc)
			v.encoding[i+1] = int(enc)
		}
	case 1:
		var nRanges uint8
		tt.read(&nRanges)
		panic("not implemented yet: encoding format 1")

	default:
		panic(fmt.Sprintf("not implemented yet: encoding format %d", version))
	}
}

type encoding struct {
	char int
	enc  int
}

// encodingSortByChar sorts encodings
type encodingSortByChar []encoding

func (a encodingSortByChar) Len() int           { return len(a) }
func (a encodingSortByChar) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a encodingSortByChar) Less(i, j int) bool { return a[i].char < a[j].char }

func (tt *Font) cffWriteEncoding(w io.Writer, v *CFFFont) int {
	if v.encodingOffset == 0 {
		return 0
	}
	version := uint8(0)
	tt.write(w, version)
	enc := make(encodingSortByChar, 0)
	for c, e := range v.encoding {
		enc = append(enc, encoding{char: c, enc: e})
	}
	sort.Sort(enc)
	tt.write(w, uint8(len(enc)))
	for _, e := range enc {
		cc := uint8(e.char)
		tt.write(w, cc)
	}
	return 2 + len(enc)
}

func (tt *Font) cffWriteCharStringIndex(w io.Writer, v *CFFFont) int {
	return tt.cffWriteIndex(w, v.charString)
}

func cffDictEncodeFloat(num float64) []byte {
	if math.Abs(float64(int(num))-num) < 0.0001 {
		return cffDictEncodeNumber(int(num))
	}

	beforeDecimal := true
	nibbles := []uint8{}
	cleandString := removeTrailingZeros.ReplaceAllString(fmt.Sprintf("%e", num), "$1")
	for _, c := range cleandString {
		if c == '-' {
			if beforeDecimal {
				nibbles = append(nibbles, 0xe)
			} else {
				// instead of 0xb
				nibbles[len(nibbles)-1] = 0xc
			}
		} else if c == '+' {
			// ignore
		} else if c == '.' {
			nibbles = append(nibbles, 0xa)
			beforeDecimal = false
		} else if c == 'e' {
			nibbles = append(nibbles, 0xb)
			beforeDecimal = false
		} else if c >= '0' && c <= '9' {
			nibbles = append(nibbles, uint8(rune(c)-'0'))
		} else {
			panic("invalid float")
		}
	}
	ret := []byte{30}
	for i := 0; i < len(nibbles)/2; i++ {
		b := nibbles[i*2] << 4
		b += nibbles[i*2+1]
		ret = append(ret, b)
		// at end, if len(nibbles) %2 != 0:
		if (i+1)*2+1 == len(nibbles) {
			b := nibbles[(i+1)*2]<<4 + 0xf
			ret = append(ret, b)
		}
	}
	if len(nibbles)%2 == 0 {
		ret = append(ret, 0xff)
	}
	return ret
}

func cffDictEncodeNumber(num int) []byte {
	if num >= -107 && num <= 107 {
		return []byte{byte(num) + 139}
	} else if num >= 108 && num <= 1131 {
		b1 := uint8(num - 108&0xff)
		b0 := uint8((num >> 8) + 247)
		return []byte{b0, b1}
	} else if num >= -1131 && num <= -108 {
		num *= -1
		b1 := uint8(num&0xff + 0xff - 108 + 1)
		b0 := uint8((num >> 8) + 251)
		return []byte{b0, b1}
	} else if num >= -32768 && num <= 32767 {
		b1 := uint8(num >> 8)
		b2 := uint8(num & 0xff)
		return []byte{28, b1, b2}
	} else if num >= -2<<31 && num <= 2<<32-1 {
		b1 := uint8(num >> 24)
		b2 := uint8(num >> 16)
		b3 := uint8(num >> 8)
		b4 := uint8(num & 0xff)
		return []byte{29, b1, b2, b3, b4}
	}
	return []byte{}
}

// cffEncodeDict returns a byte slice of the encoded dictionary
func (tt *Font) cffEncodeDict(c *CFFFont) []byte {
	var b []byte
	if c.version != "" {
		b = append(b, cffDictEncodeNumber(tt.CFF.stringToInt[c.version])...)
		b = append(b, 0)
	}
	if str := c.notice; str != "" {
		b = append(b, cffDictEncodeNumber(tt.CFF.stringToInt[str])...)
		b = append(b, 1)
	}
	if str := c.fullname; str != "" {
		b = append(b, cffDictEncodeNumber(tt.CFF.stringToInt[str])...)
		b = append(b, 2)
	}
	if str := c.familyname; str != "" {
		b = append(b, cffDictEncodeNumber(tt.CFF.stringToInt[str])...)
		b = append(b, 3)
	}
	if str := c.weight; str != "" {
		b = append(b, cffDictEncodeNumber(tt.CFF.stringToInt[str])...)
		b = append(b, 4)
	}
	if num := c.uniqueid; num != 0 {
		b = append(b, cffDictEncodeNumber(num)...)
		b = append(b, 13)
	}
	if c.bbox[0] != 0 || c.bbox[1] != 0 || c.bbox[2] != 0 || c.bbox[3] != 0 {
		b = append(b, cffDictEncodeNumber(c.bbox[0])...)
		b = append(b, cffDictEncodeNumber(c.bbox[1])...)
		b = append(b, cffDictEncodeNumber(c.bbox[2])...)
		b = append(b, cffDictEncodeNumber(c.bbox[3])...)
		b = append(b, 5)
	}
	if num := c.underlinePosition; num != -100 {
		b = append(b, cffDictEncodeFloat(num)...)
		b = append(b, 12, 3)
	}
	if num := c.underlineThickness; num != 50 {
		b = append(b, cffDictEncodeFloat(num)...)
		b = append(b, 12, 4)
	}
	if num := c.charsetOffset; num != 0 {
		b = append(b, cffDictEncodeNumber(int(num))...)
		b = append(b, 15)
	}
	if num := c.encodingOffset; num != 0 {
		b = append(b, cffDictEncodeNumber(num)...)
		b = append(b, 16)
	}
	if num := c.charstringsOffset; num != 0 {
		b = append(b, cffDictEncodeNumber(int(num))...)
		b = append(b, 17)
	}
	if num := c.privatedictoffset; num != 0 {
		b = append(b, cffDictEncodeNumber(c.privatedictsize)...)
		b = append(b, cffDictEncodeNumber(int(num))...)
		b = append(b, 18)
	}
	return b
}

// Delegate to top CFF object
func (c *CFFFont) readString(sid int) string {
	return c.global.readString(sid)
}

// Top DICT Data - see CFF spec 9 p. 14
func (c *CFFFont) parseDict(dict []byte) {
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
			c.version = c.readString(operands[0])
			operands = operands[:0]
		} else if b0 == 1 {
			// notice
			c.notice = c.readString(operands[0])
			operands = operands[:0]
		} else if b0 == 2 {
			// fullname
			c.fullname = c.readString(operands[0])
			operands = operands[:0]
		} else if b0 == 3 {
			c.familyname = c.readString(operands[0])
			operands = operands[:0]
		} else if b0 == 4 {
			// weight
			c.weight = c.readString(operands[0])
			operands = operands[:0]
		} else if b0 == 5 {
			// font bbox
			c.bbox = make([]int, 4)
			copy(c.bbox, operands)
			operands = operands[:0]
		} else if b0 == 6 {
			// Blue Values
			c.bluevalues = parseDelta(operands)
			operands = operands[:0]
		} else if b0 == 7 {
			c.otherblues = parseDelta(operands)
			operands = operands[:0]
		} else if b0 == 8 {
			c.familyblues = parseDelta(operands)
			operands = operands[:0]
		} else if b0 == 9 {
			c.familyotherblues = parseDelta(operands)
			operands = operands[:0]
		} else if b0 == 10 {
			c.stdhw = operands[0]
			operands = operands[:0]
		} else if b0 == 11 {
			c.stdvw = operands[0]
			operands = operands[:0]
		} else if b0 == 12 {
			// two bytes
			pos++
			b1 := dict[pos]
			switch b1 {
			case 3:
				if len(operands) > 0 {
					c.underlinePosition = float64(operands[0])
				} else if len(operandsf) > 0 {
					c.underlinePosition = operandsf[0]
				}
			case 4:
				if len(operands) > 0 {
					c.underlineThickness = float64(operands[0])
				} else if len(operandsf) > 0 {
					c.underlineThickness = operandsf[0]
				}
			case 8:
				// StrokeWidth
			case 9:
				c.bluescale = operandsf[0]
			// case 10:
			// 	c.blueshift = operands[0]
			case 11:
				c.bluefuzz = operands[0]
			case 12:
				c.stemsnaph = parseDelta(operands)
			case 13:
				c.stemsnapv = parseDelta(operands)
			case 30:
				// ROS
				c.registry = c.readString(operands[0])
				c.ordering = c.readString(operands[1])
				c.supplement = operands[2]
			case 34:
				// CID count
				c.cidcount = operands[0]
			case 36:
				// FDArray
				c.fdarray = int64(operands[0])
			case 37:
				// FDSelect
				c.fdselect = int64(operands[0])
			case 38:
				// fontname
				c.name = c.readString(operands[0])
			default:
				panic(fmt.Sprintf("not implemented ESC %d", b1))
			}
			operands = operands[:0]
			operandsf = operandsf[:0]
		} else if b0 == 13 {
			// unique id
			c.uniqueid = operands[0]
			operands = operands[:0]
		} else if b0 == 15 {
			// charset
			c.charsetOffset = int64(operands[0])
			operands = operands[:0]
		} else if b0 == 16 {
			c.encodingOffset = operands[0]
			operands = operands[:0]
		} else if b0 == 17 {
			// charstrings (type 2 instructions)
			c.charstringsOffset = int64(operands[0])
			operands = operands[:0]
		} else if b0 == 18 {
			c.privatedictsize = operands[0]
			c.privatedictoffset = int64(operands[1])
		} else if b0 == 19 {
			// initialRandomSeed ?? subrs?
			c.initialRandomSeed = int(operands[0])
			operands = operands[:0]
		} else if b0 == 21 {
			c.nominalWidthX = operands[0]
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
			shift := 0
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
			flt += float64(valafter) / float64(div)
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

// IsCIDFont returns true if the character encoding is based on CID instead of SID
func (c *CFFFont) IsCIDFont() bool {
	return c.fdselect != 0
}

func (c *CFF) readString(sid int) string {
	return c.strings[sid]
}

func (tt *Font) readCFF(tbl tableOffsetLength) error {
	saveR := tt.r
	defer func() {
		tt.r = saveR
	}()
	var err error
	tbl.tabledata, err = tt.ReadTableData("CFF ")
	if err != nil {
		return err
	}
	tt.r = bytes.NewReader(tbl.tabledata)
	tt.CFF = CFF{}

	tt.CFF.strings = make([]string, len(predefinedStrings))
	tt.CFF.stringToInt = make(map[string]int, len(predefinedStrings))

	for i, v := range predefinedStrings {
		tt.CFF.strings[i] = v
		tt.CFF.stringToInt[v] = i
	}

	tt.read(&tt.CFF.Major)
	tt.read(&tt.CFF.Minor)
	tt.read(&tt.CFF.HdrSize)
	tt.read(&tt.CFF.offsetSize)

	tt.cffReadNameIndex()
	tt.cffReadDictIndex()
	tt.cffReadStringIndex()
	tt.cffReadGlobalSubrIndex()
	for _, v := range tt.CFF.font {
		v.parseDict(v.dict)
		v.dict = nil
		tt.cffReadEncoding(v)
		tt.cffReadCharstring(v)
		tt.cffReadCharset(v)
		tt.cffReadPrivateDict(v)
	}
	return nil
}

// writeCFF writes the “CFF ” table  to th given writer.
// The data structure in the cff font is:
//
// Header
// Name INDEX
// Top DICT INDEX
// String INDEX
// Global Subr INDEX
// Encodings
// Charsets
func (tt *Font) writeCFF(w io.Writer) error {
	tt.write(w, tt.CFF.Major)
	tt.write(w, tt.CFF.Minor)
	tt.write(w, tt.CFF.HdrSize)
	tt.write(w, tt.CFF.offsetSize)
	cffLen := 4
	cffLen += tt.cffWriteNameIndex(w)
	cffLen += tt.cffWriteDictIndex(w)
	cffLen += tt.cffWriteStringIndex(w)
	cffLen += tt.cffWriteGlobalSubrIndex(w)
	// charset?
	for _, v := range tt.CFF.font {
		cffLen += tt.cffWriteCharset(w, v)
		cffLen += tt.cffWriteEncoding(w, v)
		cffLen += tt.cffWriteCharStringIndex(w, v)
		cffLen += tt.cffWritePrivateDict(w, v)
	}
	return nil
}
