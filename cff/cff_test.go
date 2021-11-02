package cff

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestEncodeCffDictData(t *testing.T) {
	testdata := []struct {
		val int
		res []byte
	}{
		{0, []byte{0x8b}},
		{100, []byte{0xef}},
		{1000, []byte{0xfa, 0x7c}},
		{-1000, []byte{0xfe, 0x7c}},
		{248, []byte{0xf7, 0x8c}},
		{600, []byte{248, 0xec}},
		{-274, []byte{251, 166}},
		{10000, []byte{0x1c, 0x27, 0x10}},
		{-10000, []byte{0x1c, 0xd8, 0xf0}},
		{100000, []byte{0x1d, 0x00, 0x01, 0x86, 0xa0}},
		{-100000, []byte{0x1d, 0xff, 0xfe, 0x79, 0x60}},
	}
	for _, td := range testdata {
		if ret := cffDictEncodeNumber(int64(td.val)); bytes.Compare(ret, td.res) != 0 {
			t.Errorf("cffDictEncodeNumber(%d) = %x, want %x", td.val, ret, td.res)
		}
	}

	testdataf := []struct {
		val float64
		res []byte
	}{
		{0, []byte{0x8b}}, // non-float
		{-0.005, []byte{0x1e, 0xe5, 0xc3, 0xff}},
		// {-0.025, []byte{0x1e, 0xe2, 0xa5, 0xc2, 0xff}},
		{-0.025, []byte{0x1e, 0xe0, 0xa0, 0x25, 0xff}},
		{25.73, []byte{0x1e, 0x2a, 0x57, 0x3b, 0x1f}},
	}
	for _, td := range testdataf {
		if ret := cffDictEncodeFloat(td.val); bytes.Compare(ret, td.res) != 0 {
			t.Errorf("cffDictEncodeFloat(%f) = %x, want %x", td.val, ret, td.res)
		}
	}
}

func TestSubsetMD(t *testing.T) {
	r, err := os.Open("testdata/maziusdisplay.cff")
	if err != nil {
		t.Error(err)
	}
	cffFontFile, err := ParseCFFData(r)
	if err != nil {
		t.Error(err)
	}
	cffFontFile.Subset([]int{0, 100, 93, 108, 115})

}

func TestCompareTables(t *testing.T) {
	r, err := os.Open("testdata/maziusdisplay.cff")
	if err != nil {
		t.Error(err)
	}
	cffFontFile, err := ParseCFFData(r)
	if err != nil {
		t.Error(err)
	}
	if got, want := cffFontFile.Major, 1; int(got) != want {
		t.Errorf("fnt.Major: %d, want %d", got, want)
	}
	if got, want := cffFontFile.Minor, 0; int(got) != want {
		t.Errorf("fnt.Minor: %d, want %d", got, want)
	}

	r.Seek(4, io.SeekStart)
	var w bytes.Buffer
	var bytesWritten int
	for _, index := range []mainIndex{NameIndex, DictIndex, StringIndex, GlobalSubrIndex} {
		w.Reset()
		origTblBytes, err := cffFontFile.GetRawIndexData(r, index)
		if err != nil {
			t.Error(err)
		}

		bytesWritten, err = cffFontFile.writeIndex(&w, index)
		if err != nil {
			t.Error(err)
		}

		if expected, got := len(origTblBytes), bytesWritten; expected != got {
			t.Errorf("bytes written = %d, want %d", got, expected)
		}

		if expected, got := len(origTblBytes), w.Len(); expected != got {
			t.Errorf("len(read) = %d, want %d", got, expected)
		}

		if cmp := bytes.Compare(origTblBytes, w.Bytes()); cmp != 0 {
			t.Errorf("compare = %d, want 0 ", cmp)
			fmt.Println(origTblBytes)
			fmt.Println(w.Bytes())
		}
	}
	for _, cf := range cffFontFile.Font {
		for _, index := range []mainIndex{CharStringsIndex, CharSet, PrivateDict, LocalSubrsIndex} {
			w.Reset()
			origTblBytes, err := cf.GetRawIndexData(r, index)
			if err != nil {
				t.Error(err)
			}
			if index == PrivateDict {
				cf.readSubrIndex(r)
			}
			bytesWritten, err = cf.writeIndex(&w, index)
			if err != nil {
				t.Error(err)
			}

			if expected, got := len(origTblBytes), bytesWritten; expected != got {
				t.Errorf("bytes written = %d, want %d", got, expected)
			}

			if expected, got := len(origTblBytes), w.Len(); expected != got {
				t.Errorf("len(read) = %d, want %d", got, expected)
			}

			if got, cmp := w.Bytes(), bytes.Compare(origTblBytes, w.Bytes()); cmp != 0 {
				t.Errorf("compare = %d, want 0 ", cmp)
				maxOrig := 20
				maxGot := 20
				if l := len(origTblBytes); l < maxOrig {
					maxOrig = l
				}
				if l := len(got); l < maxGot {
					maxGot = l
				}
				fmt.Println(origTblBytes[:maxOrig])
				fmt.Println(got[:maxGot])
			}
		}
	}
}

func TestCompareTablesFST(t *testing.T) {
	r, err := os.Open("testdata/firasansthin.cff")
	if err != nil {
		t.Error(err)
	}
	cffFontFile, err := ParseCFFData(r)
	if err != nil {
		t.Error(err)
	}
	if got, want := cffFontFile.Major, 1; int(got) != want {
		t.Errorf("fnt.Major: %d, want %d", got, want)
	}
	if got, want := cffFontFile.Minor, 0; int(got) != want {
		t.Errorf("fnt.Minor: %d, want %d", got, want)
	}

	r.Seek(4, io.SeekStart)
	var w bytes.Buffer
	var bytesWritten int
	for _, index := range []mainIndex{NameIndex, DictIndex, StringIndex, GlobalSubrIndex} {
		w.Reset()
		origTblBytes, err := cffFontFile.GetRawIndexData(r, index)
		if err != nil {
			t.Error(err)
		}

		bytesWritten, err = cffFontFile.writeIndex(&w, index)
		if err != nil {
			t.Error(err)
		}

		if expected, got := len(origTblBytes), bytesWritten; expected != got {
			t.Errorf("bytes written = %d, want %d", got, expected)
		}

		if expected, got := len(origTblBytes), w.Len(); expected != got {
			t.Errorf("len(read) = %d, want %d", got, expected)
		}

		if cmp := bytes.Compare(origTblBytes, w.Bytes()); cmp != 0 {
			t.Errorf("compare = %d, want 0 ", cmp)
			fmt.Println(origTblBytes)
			fmt.Println(w.Bytes())
		}
	}
	for _, cf := range cffFontFile.Font {
		for _, index := range []mainIndex{CharStringsIndex, CharSet, PrivateDict, LocalSubrsIndex} {
			w.Reset()
			origTblBytes, err := cf.GetRawIndexData(r, index)
			if err != nil {
				t.Error(err)
			}
			if index == PrivateDict {
				cf.readSubrIndex(r)
			}
			bytesWritten, err = cf.writeIndex(&w, index)
			if err != nil {
				t.Error(err)
			}

			if expected, got := len(origTblBytes), bytesWritten; expected != got {
				t.Errorf("bytes written = %d, want %d", got, expected)
			}

			if expected, got := len(origTblBytes), w.Len(); expected != got {
				t.Errorf("len(read) = %d, want %d", got, expected)
			}

			if got, cmp := w.Bytes(), bytes.Compare(origTblBytes, w.Bytes()); cmp != 0 {
				t.Errorf("compare = %d, want 0 ", cmp)
				maxOrig := 20
				maxGot := 20
				if l := len(origTblBytes); l < maxOrig {
					maxOrig = l
				}
				if l := len(got); l < maxGot {
					maxGot = l
				}
				fmt.Println(origTblBytes[:maxOrig])
				fmt.Println(got[:maxGot])
			}
		}
	}
}

func TestCompareTablesCustomFont(t *testing.T) {
	r, err := os.Open("testdata/customfont.cff")
	if err != nil {
		t.Error(err)
	}
	cffFontFile, err := ParseCFFData(r)
	if err != nil {
		t.Error(err)
	}
	if got, want := cffFontFile.Major, 1; int(got) != want {
		t.Errorf("fnt.Major: %d, want %d", got, want)
	}
	if got, want := cffFontFile.Minor, 0; int(got) != want {
		t.Errorf("fnt.Minor: %d, want %d", got, want)
	}

	r.Seek(4, io.SeekStart)
	var w bytes.Buffer
	var bytesWritten int
	for _, index := range []mainIndex{NameIndex, DictIndex, StringIndex, GlobalSubrIndex} {
		w.Reset()
		origTblBytes, err := cffFontFile.GetRawIndexData(r, index)
		if err != nil {
			t.Error(err)
		}

		bytesWritten, err = cffFontFile.writeIndex(&w, index)
		if err != nil {
			t.Error(err)
		}

		if expected, got := len(origTblBytes), bytesWritten; expected != got {
			t.Errorf("bytes written = %d, want %d", got, expected)
		}

		if expected, got := len(origTblBytes), w.Len(); expected != got {
			t.Errorf("len(read) = %d, want %d", got, expected)
		}

		if cmp := bytes.Compare(origTblBytes, w.Bytes()); cmp != 0 {
			t.Errorf("compare = %d, want 0 ", cmp)
			fmt.Println(origTblBytes)
			fmt.Println(w.Bytes())
		}
	}
	for _, cf := range cffFontFile.Font {
		for _, index := range []mainIndex{CharStringsIndex, Encoding, CharSet, PrivateDict, LocalSubrsIndex} {
			w.Reset()
			origTblBytes, err := cf.GetRawIndexData(r, index)
			if err != nil {
				t.Error(err)
			}
			if index == PrivateDict {
				cf.readSubrIndex(r)
			}
			bytesWritten, err = cf.writeIndex(&w, index)
			if err != nil {
				t.Error(err)
			}

			if expected, got := len(origTblBytes), bytesWritten; expected != got {
				t.Errorf("bytes written = %d, want %d", got, expected)
			}

			if expected, got := len(origTblBytes), w.Len(); expected != got {
				t.Errorf("len(read) = %d, want %d", got, expected)
			}

			if got, cmp := w.Bytes(), bytes.Compare(origTblBytes, w.Bytes()); cmp != 0 {
				t.Errorf("compare = %d, want 0 ", cmp)
				maxOrig := 20
				maxGot := 20
				if l := len(origTblBytes); l < maxOrig {
					maxOrig = l
				}
				if l := len(got); l < maxGot {
					maxGot = l
				}
				fmt.Println(origTblBytes[:maxOrig])
				fmt.Println(got[:maxGot])
			}
		}
	}
}
