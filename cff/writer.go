package cff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
)

var (
	removeTrailingZeros     *regexp.Regexp
	removeTrailingZerosNonE *regexp.Regexp
	exponent                *regexp.Regexp
)

func init() {
	removeTrailingZeros = regexp.MustCompile(`\.?0*(e[+-]?)0*`)
	removeTrailingZerosNonE = regexp.MustCompile(`^0*(\.)`)
	exponent = regexp.MustCompile(`.*e-|\+(.*)$`)
}

func write(w io.Writer, data interface{}) error {
	return binary.Write(w, binary.BigEndian, data)
}

// write3uint8 writes the offset as a three byte data set.
func write3uint8(w io.Writer, offset int) error {
	data := make([]byte, 3)
	data[0] = byte(offset >> 16 & 0xff)
	data[1] = byte(offset >> 8 & 0xff)
	data[2] = byte(offset & 0xff)
	return write(w, data)
}

// writeOffset writes the offset to w which can be encoded in 1 to 4 bytes.
func writeOffset(w io.Writer, offsetsize uint8, offset int) error {
	switch offsetsize {
	case 1:
		return write(w, uint8(offset))
	case 2:
		return write(w, uint16(offset))
	case 3:
		return write3uint8(w, offset)
	case 4:
		return write(w, uint32(offset))
	default:
		panic(fmt.Sprintf("not implemented offset size %d", offsetsize))
	}
}

// writeIndexData writes the data slices to the writer w in CFF index format (cf CFF spec 5 INDEX Data p. 12).
// It returns the total number of bytes written to the writer.
func writeIndexData(w io.Writer, data [][]byte, name string) (int, error) {
	log.WithField("name", name).Trace("CFF write index")
	count := uint16(len(data))
	var err error
	err = write(w, count)
	if err != nil {
		return 0, err
	}

	if count == 0 {
		return 2, nil
	}
	indexLen := 2
	lendata := 0
	for _, b := range data {
		lendata += len(b)
	}
	log.WithField("lendata", lendata).Trace("write index")
	var offsetSize uint8
	if lendata <= 1<<8 {
		offsetSize = 1
	} else if lendata < 1<<16 {
		offsetSize = 2
	} else if lendata < 1<<24 {
		offsetSize = 3
	} else {
		offsetSize = 4
	}
	log.WithField("offsetsize", offsetSize).Trace("write")
	if err = write(w, offsetSize); err != nil {
		return 0, err
	}
	if err = writeOffset(w, offsetSize, 1); err != nil {
		return 0, err
	}
	indexLen += int(offsetSize) + 1
	for c, i := 0, 0; i < len(data); i++ {
		c += len(data[i])
		if err = writeOffset(w, offsetSize, c+1); err != nil {
			return 0, err
		}
	}
	indexLen += len(data) * int(offsetSize)
	for _, b := range data {
		if err = write(w, b); err != nil {
			return 0, err
		}
		indexLen += len(b)
	}
	return indexLen, nil
}

func (c *CFF) writeNameIndex(w io.Writer) (int, error) {
	data := make([][]byte, 0)
	for _, str := range c.fontnames {
		data = append(data, []byte(str))
	}
	return writeIndexData(w, data, "name")
}

func (c *CFF) writeDictIndex(w io.Writer) (int, error) {
	var data [][]byte
	for _, fnt := range c.Font {
		data = append(data, fnt.cffEncodeTopDict())
	}
	return writeIndexData(w, data, "dict")
}

// writeStringIndex writes all (non-predefined) strings to the writer w.
// It returns the total number of bytes written to w.
func (c *CFF) writeStringIndex(w io.Writer) (int, error) {
	var data [][]byte
	// only write the not-predefined strings
	for _, str := range c.strings[len(predefinedStrings):] {
		data = append(data, []byte(str))
	}
	return writeIndexData(w, data, "string")
}

func (c *CFF) writeGlobalSubrIndex(w io.Writer) (int, error) {
	return writeIndexData(w, c.globalSubrIndex, "global subr")
}

// writeIndex returns the number of bytes written to the index and an error.
func (c *CFF) writeIndex(w io.Writer, index mainIndex) (int, error) {
	log.WithField("index", index).Trace("writeIndex")
	switch index {
	case NameIndex:
		return c.writeNameIndex(w)
	case DictIndex:
		return c.writeDictIndex(w)
	case StringIndex:
		return c.writeStringIndex(w)
	case GlobalSubrIndex:
		return c.writeGlobalSubrIndex(w)
	default:
	}

	return 0, fmt.Errorf("Could not write index %d", index)
}

func (c *CFF) writeHeader(w io.Writer) error {
	write(w, c.Major)
	write(w, c.Minor)
	write(w, c.HdrSize)
	write(w, c.offsetSize)
	return nil
}

// WriteCFFData writes the CFF data to w.
func (c *CFF) WriteCFFData(w io.Writer) error {
	var err error
	var l int
	if err = c.writeHeader(w); err != nil {
		return err
	}

	if l, err = c.writeNameIndex(w); err != nil {
		return err
	}

	cur := 4 + l

	// We need to save the string index and the global subr index to be added
	// after the dict index.
	// The dict index needs information about offsets. These offsets need to take into
	// account the length of the string index and the global subr index.
	var stringGlobalSubrIndex bytes.Buffer
	var dictIndex bytes.Buffer

	for _, idx := range []mainIndex{StringIndex, GlobalSubrIndex} {
		_, err := c.writeIndex(&stringGlobalSubrIndex, idx)
		if err != nil {
			return err
		}
	}

	// Now let's the dict index into a temporary buffer so we know the length
	// of the buffer.
	_, err = c.writeIndex(&dictIndex, DictIndex)
	if err != nil {
		return err
	}

	for _, cf := range c.Font {
		fi, err := cf.fontInfo()
		if err != nil {
			return err
		}
		// let's assume one font only for now
		// offsets are now header + name index + len(dictindex) + len(string index) + len(global subr index) + offsets
		// that is                         cur + len(dictindex)  + stringGlobalSubrIndex.Len() + offsets
		baselen := cur + dictIndex.Len() + stringGlobalSubrIndex.Len()

		// encodings can be ignored
		cf.encodingOffset = 0
		cf.charstringsOffset = int64(baselen + fi.CharStringsOffset)
		cf.charsetOffset = int64(baselen + fi.CharSetOffset)
		cf.privatedictoffset = int64(baselen + fi.PrivateDictOffset)
		cf.privatedictsize = fi.PrivateDictSize
	}

	// now we can write all data
	// header + NameIndex is already written to w
	_, err = c.writeIndex(w, DictIndex)
	if err != nil {
		return err
	}
	// The pre created string index and the global subr index
	stringGlobalSubrIndex.WriteTo(w)

	// And now for each font the char set, char string, private dict and local subr index.
	// The data field is created in fontInfo() above.
	for _, cf := range c.Font {
		_, err = w.Write(cf.data)
		if err != nil {
			return err
		}
	}
	return nil
}

type fontinfo struct {
	CharSetOffset     int
	CharStringsOffset int
	EncodingOffset    int
	PrivateDictSize   int
	PrivateDictOffset int
}

func (f *Font) fontInfo() (*fontinfo, error) {
	fi := &fontinfo{}
	fi.CharSetOffset = 0
	var b bytes.Buffer

	for _, index := range []mainIndex{CharSet, CharStringsIndex, PrivateDict, LocalSubrsIndex} {
		switch index {
		case CharStringsIndex:
			fi.CharStringsOffset = b.Len()
		case PrivateDict:
			fi.PrivateDictOffset = b.Len()
		case Encoding:
			fi.EncodingOffset = b.Len()
		case LocalSubrsIndex:
			cur := b.Len()
			fi.PrivateDictSize = cur - fi.PrivateDictOffset
		}
		if index != LocalSubrsIndex || len(f.subrsIndex) > 0 {
			if _, err := f.writeIndex(&b, index); err != nil {
				return nil, err
			}
		}

	}
	f.data = b.Bytes()
	return fi, nil
}

// cffDictEncodeFloat encodes a number. If the number is an integer number, it will be encoded by cffDictEncodeNumber().
func cffDictEncodeFloat(num float64) []byte {
	if math.Abs(float64(int(num))-num) < 0.0001 {
		return cffDictEncodeNumber(int(num))
	}

	beforeDecimal := true
	nibbles := []uint8{}

	var cleanedString string

	// if the exponent is < 3, use the 0.00xx notation instead of e+x
	cleanedString = removeTrailingZeros.ReplaceAllString(fmt.Sprintf("%e", num), "$1")
	if exp, err := strconv.Atoi(exponent.ReplaceAllString(cleanedString, "$1")); err == nil && exp < 3 {
		cleanedString = removeTrailingZerosNonE.ReplaceAllString(fmt.Sprintf("%g", num), "$1")
	}

	for _, c := range cleanedString {
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
		num = num - 108

		b1 := uint8(num & 0xff)
		b0 := uint8((num >> 8) + 247)
		return []byte{b0, b1}
	} else if num >= -1131 && num <= -108 {
		num += 108
		num *= -1
		b1 := uint8(num & 0xff)
		b0 := uint8(num>>8) + 251
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

// cffEncodeTopDict returns a byte slice of the encoded dictionary
func (f *Font) cffEncodeTopDict() []byte {
	var b []byte
	if i := f.version; i != 0 {
		log.WithField("fnt.version", f.version).Trace("dict write version")
		b = append(b, cffDictEncodeNumber(int(i))...)
		b = append(b, 0)
	}
	if i := f.notice; i != 0 {
		log.Trace("dict write notice")
		b = append(b, cffDictEncodeNumber(int(i))...)
		b = append(b, 1)
	}
	if i := f.copyright; i != 0 {
		log.Trace("dict write copyright")
		b = append(b, cffDictEncodeNumber(int(i))...)
		b = append(b, 12, 0)
	}
	if i := f.fullname; i != 0 {
		log.Trace("dict write full name")
		b = append(b, cffDictEncodeNumber(int(i))...)
		b = append(b, 2)
	}
	if i := f.familyname; i != 0 {
		log.Trace("dict write family name")
		b = append(b, cffDictEncodeNumber(int(i))...)
		b = append(b, 3)
	}
	if i := f.weight; i != 0 {
		log.Trace("dict write weight")
		b = append(b, cffDictEncodeNumber(int(i))...)
		b = append(b, 4)
	}
	if num := f.uniqueid; num != 0 {
		log.Trace("dict write uid")
		b = append(b, cffDictEncodeNumber(num)...)
		b = append(b, 13)
	}
	if f.bbox[0] != 0 || f.bbox[1] != 0 || f.bbox[2] != 0 || f.bbox[3] != 0 {
		log.Trace("dict write font bbox")
		b = append(b, cffDictEncodeNumber(f.bbox[0])...)
		b = append(b, cffDictEncodeNumber(f.bbox[1])...)
		b = append(b, cffDictEncodeNumber(f.bbox[2])...)
		b = append(b, cffDictEncodeNumber(f.bbox[3])...)
		b = append(b, 5)
	}
	if num := f.underlinePosition; num != -100 {
		log.Trace("dict write underline position")
		b = append(b, cffDictEncodeFloat(num)...)
		b = append(b, 12, 3)
	}
	if num := f.underlineThickness; num != 50 {
		log.Trace("dict write underline thickness")
		b = append(b, cffDictEncodeFloat(num)...)
		b = append(b, 12, 4)
	}
	if num := f.charsetOffset; num != 0 {
		log.WithField("off", num).Trace("dict write charsetoffset")
		b = append(b, cffDictEncodeNumber(int(num))...)
		b = append(b, 15)
	}
	if num := f.encodingOffset; num != 0 {
		log.WithField("offset", num).Trace("dict write encodingoffset")
		b = append(b, cffDictEncodeNumber(num)...)
		b = append(b, 16)
	}
	if num := f.charstringsOffset; num != 0 {
		log.Trace("dict write charstring offset")
		b = append(b, cffDictEncodeNumber(int(num))...)
		b = append(b, 17)
	}
	if num := f.privatedictoffset; num != 0 {
		log.Trace("dict write privatedictsize, offset")
		b = append(b, cffDictEncodeNumber(f.privatedictsize)...)
		b = append(b, cffDictEncodeNumber(int(num))...)
		b = append(b, 18)
	}
	return b
}

// cffEncodePrivateDict returns a byte slice of the encoded dictionary
func (f *Font) cffEncodePrivateDict() []byte {
	var b []byte
	if len(f.bluevalues) > 0 {
		log.WithField("val", f.bluevalues).Trace("dict write bluevalues")
		for _, v := range f.bluevalues {
			b = append(b, cffDictEncodeNumber(v)...)
		}
		b = append(b, 6)
	}
	if len(f.otherblues) > 0 {
		log.WithField("val", f.otherblues).Trace("dict write otherblues")
		for _, v := range f.otherblues {
			b = append(b, cffDictEncodeNumber(v)...)
		}
		b = append(b, 7)
	}
	if len(f.familyblues) > 0 {
		log.WithField("val", f.familyblues).Trace("dict write familyblues")
		for _, v := range f.familyblues {
			b = append(b, cffDictEncodeNumber(v)...)
		}
		b = append(b, 8)
	}
	if len(f.familyotherblues) > 0 {
		log.WithField("val", f.familyotherblues).Trace("dict write familyotherblues")
		for _, v := range f.familyotherblues {
			b = append(b, cffDictEncodeNumber(v)...)
		}
		b = append(b, 9)
	}

	if num := f.bluescale; num != 0.039625 {
		log.WithField("num", num).Trace("dict write blue scale")
		b = append(b, cffDictEncodeFloat(num)...)
		b = append(b, 12, 9)
	}
	if num := f.bluefuzz; num != 1 {
		log.WithField("num", num).Trace("dict write blue fuzz")
		b = append(b, cffDictEncodeFloat(float64(f.bluefuzz))...)
		b = append(b, 12, 11)
	}
	if num := f.stdhw; num != 0 {
		log.WithField("num", num).Trace("dict write stdhw")
		b = append(b, cffDictEncodeFloat(float64(f.stdhw))...)
		b = append(b, 10)
	}
	if num := f.stdvw; num != 0 {
		log.WithField("num", num).Trace("dict write stdvw")
		b = append(b, cffDictEncodeFloat(float64(f.stdvw))...)
		b = append(b, 11)
	}
	if len(f.stemsnaph) > 0 {
		log.WithField("val", f.stemsnaph).Trace("dict write stemsnaph")
		for _, v := range f.stemsnaph {
			b = append(b, cffDictEncodeNumber(v)...)
		}
		b = append(b, 12, 12)
	}
	if len(f.stemsnapv) > 0 {
		log.WithField("val", f.stemsnapv).Trace("dict write stemsnapv")
		for _, v := range f.stemsnapv {
			b = append(b, cffDictEncodeNumber(v)...)
		}
		b = append(b, 12, 13)
	}
	if num := f.defaultWidthX; num != 0 {
		log.WithField("defaultWidthX", num).Trace("dict write defaultWidthX")
		b = append(b, cffDictEncodeFloat(float64(num))...)
		b = append(b, 20)
	}
	if num := f.nominalWidthX; num != 0 {
		log.WithField("nominalWidthX", num).Trace("dict write nominalWidthX")
		b = append(b, cffDictEncodeFloat(float64(num))...)
		b = append(b, 21)
	}
	if len(f.subrsIndex) > 0 {
		log.WithField("off", len(b)+2).Trace("dict write local subrs offset")
		b = append(b, cffDictEncodeNumber(len(b)+2)...)
		b = append(b, 19)
	}
	return b
}

func (f *Font) writeCharStringsIndex(w io.Writer) (int, error) {
	return writeIndexData(w, f.CharStrings, "charstrings")
}

func (f *Font) writeCharSet(w io.Writer) (int, error) {
	log.WithField("fmt", f.charsetFormat).Trace("Write CharSet")
	var err error

	write(w, f.charsetFormat)

	switch f.charsetFormat {
	case 0:
		var sid uint16
		for i := 1; i < len(f.CharStrings); i++ {
			sid = uint16(f.charset[i])
			if err = write(w, sid); err != nil {
				return 0, err
			}
		}
		return 1 + (len(f.CharStrings)-1)*2, nil
	case 1:
		var sid uint16
		c := 1
		cur := 0
		i := 1

		// f.charset[0] is notdef, we skip that
		cs := f.charset[1:]
		for {
			sid = uint16(cs[cur])
			if err = write(w, sid); err != nil {
				return 0, err
			}
			c += 2
			glyphsLeft := uint8(0)
		inner:
			for {
				// glyphsLeft is uint8, we need to check < 255
				if cur+i < len(cs) && cs[cur+i] == SID(sid)+SID(i) && glyphsLeft < 255 {
					glyphsLeft++
					i++
				} else {
					if err = write(w, glyphsLeft); err != nil {
						return 0, err
					}
					c++
					cur = cur + i
					i = 1
					break inner
				}
			}

			if cur >= len(cs) {
				break
			}
		}

		return c, nil
	}
	return 0, nil
}

func (f *Font) writePrivateDict(w io.Writer) (int, error) {
	f.privatedict = f.cffEncodePrivateDict()
	err := write(w, f.privatedict)
	return len(f.privatedict), err
}

func (f *Font) writeLocalSubrsIndex(w io.Writer) (int, error) {
	if len(f.subrsIndex) == 0 {
		return 0, nil
	}
	return writeIndexData(w, f.subrsIndex, "subrIndex")
}

func (f *Font) writeEncoding(w io.Writer) (int, error) {
	var err error
	if err = write(w, uint8(0)); err != nil {
		return 0, err
	}

	if err = write(w, uint8(len(f.encoding))); err != nil {
		return 0, err
	}

	for i := 1; i <= len(f.encoding); i++ {
		if err = write(w, uint8(f.encoding[i])); err != nil {
			return 0, err
		}
	}
	return 2 + len(f.encoding), nil
}

// writeIndex returns the number of bytes written to the index and an error.
func (f *Font) writeIndex(w io.Writer, index mainIndex) (int, error) {
	switch index {
	case CharStringsIndex:
		return f.writeCharStringsIndex(w)
	case CharSet:
		return f.writeCharSet(w)
	case Encoding:
		return f.writeEncoding(w)
	case PrivateDict:
		return f.writePrivateDict(w)
	case LocalSubrsIndex:
		return f.writeLocalSubrsIndex(w)
	default:
	}

	return 0, fmt.Errorf("Could not write index %d", index)
}
