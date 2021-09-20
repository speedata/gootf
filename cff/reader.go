package cff

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.SetFormatter(&nested.Formatter{
		HideKeys: false,
		NoColors: true,
	})

	log.SetLevel(logrus.TraceLevel)
	// log.SetReportCaller(true)
	log.SetOutput(os.Stdout)
}

func read(r io.Reader, data interface{}) error {
	return binary.Read(r, binary.BigEndian, data)
}

func read3uint32(r io.Reader) uint32 {
	data := make([]byte, 3)
	r.Read(data)
	var res uint32
	res = uint32(data[0])<<16 + uint32(data[1])<<8 + uint32(data[2])
	return res
}

func readBytes(r io.Reader, n int) []byte {
	b := make([]byte, n)
	l, err := r.Read(b)
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
func readOffset(r io.Reader, offsetsize uint8) uint32 {
	switch offsetsize {
	case 1:
		var offset uint8
		read(r, &offset)
		return uint32(offset)
	case 2:
		var offset uint16
		read(r, &offset)
		return uint32(offset)
	case 3:
		return read3uint32(r)
	case 4:
		var offset uint32
		read(r, &offset)
		return offset
	default:
		panic(fmt.Sprintf("not implemented offset size %d", offsetsize))
	}
}

// cffReadIndexData reads a number of slices with data.
func cffReadIndexData(r io.Reader, name string) [][]byte {
	// log.WithField("idx", name).Trace("CFF read index")
	var count uint16
	read(r, &count)
	if count == 0 {
		return [][]byte{}
	}
	var offsetSize uint8
	read(r, &offsetSize)
	offsets := make([]int, count+1)
	for i := 0; i < int(count+1); i++ {
		offsets[i] = int(readOffset(r, offsetSize))
	}
	data := make([][]byte, count)
	for i := 0; i < int(count); i++ {
		data[i] = readBytes(r, offsets[i+1]-offsets[i])
	}
	return data
}

// an array a0, a1, ..., an would be encoded as
// a0 (a1–a0) (a2–a1) ..., (an–a(n–1))
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

func (c *CFF) readNameIndex(r io.Reader) error {
	idx := cffReadIndexData(r, "name")
	c.fontnames = []string{}
	for _, entry := range idx {
		log.WithField("name", string(entry)).Trace("read name")
		c.fontnames = append(c.fontnames, string(entry))
	}
	return nil
}

func (c *CFF) readDictIndex(r io.Reader) error {
	c.Font = []*Font{}
	allFonts := cffReadIndexData(r, "dict")
	for _, cffFont := range allFonts {
		fnt := &Font{
			underlineThickness: 50,
			underlinePosition:  -100,
		}
		fnt.parseDict(cffFont)
		c.Font = append(c.Font, fnt)
	}
	return nil
}

func (c *CFF) readStringIndex(r io.Reader) error {
	c.initStrings()
	si := cffReadIndexData(r, "string")
	for _, entry := range si {
		str := string(entry)
		c.strings = append(c.strings, str)
		c.stringToInt[str] = len(c.strings) - 1
	}
	return nil
}

func (c *CFF) readGlobalSubrIndex(r io.Reader) error {
	subr := cffReadIndexData(r, "subr")
	c.globalSubrIndex = [][]byte{}
	for _, entry := range subr {
		c.globalSubrIndex = append(c.globalSubrIndex, entry)
	}

	return nil
}

// GetRawIndexData returns a byte slice of the index
func (c *CFF) GetRawIndexData(r io.ReadSeeker, index mainIndex) ([]byte, error) {
	indexStart, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	switch index {
	case NameIndex:
		err = c.readNameIndex(r)
	case DictIndex:
		err = c.readDictIndex(r)
	case StringIndex:
		err = c.readStringIndex(r)
	case GlobalSubrIndex:
		err = c.readGlobalSubrIndex(r)
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
func (c *CFF) parseIndex(r io.ReadSeeker, index mainIndex) error {
	var err error
	switch index {
	case NameIndex:
		err = c.readNameIndex(r)
	case DictIndex:
		err = c.readDictIndex(r)
	case StringIndex:
		err = c.readStringIndex(r)
	case GlobalSubrIndex:
		err = c.readGlobalSubrIndex(r)
	default:
		panic(fmt.Sprintf("unknown index %d", index))
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *CFF) initStrings() {
	c.strings = make([]string, len(predefinedStrings))
	c.stringToInt = make(map[string]int, len(predefinedStrings))
	for i, v := range predefinedStrings {
		c.strings[i] = v
		c.stringToInt[v] = i
	}
}

// FontName returns the PostScript font name of the font to be written
func (c *CFF) FontName() string {
	return c.fontnames[0]
}

// ParseCFFData interprets the CFF data and returns an error or nil.
func ParseCFFData(r io.ReadSeeker) (*CFF, error) {
	log.Debug("ParseCFFData")
	cff := &CFF{}

	read(r, &cff.Major)
	read(r, &cff.Minor)
	read(r, &cff.HdrSize)
	read(r, &cff.offsetSize)
	r.Seek(int64(cff.HdrSize), io.SeekStart)
	if err := cff.parseIndex(r, NameIndex); err != nil {
		return nil, err
	}
	if err := cff.parseIndex(r, DictIndex); err != nil {
		return nil, err
	}
	if err := cff.parseIndex(r, StringIndex); err != nil {
		return nil, err
	}
	if err := cff.parseIndex(r, GlobalSubrIndex); err != nil {
		return nil, err
	}

	for _, fnt := range cff.Font {
		fnt.parseIndex(r, CharStringsIndex)
		fnt.parseIndex(r, Encoding)
		fnt.parseIndex(r, CharSet)
		fnt.parseIndex(r, PrivateDict)
		if fnt.subrsOffset > 0 {
			fnt.parseIndex(r, LocalSubrsIndex)
		}
	}

	log.Debug("ParseCFFData...done")
	return cff, nil
}
