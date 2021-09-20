package cff

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

// func TestGetSubrs(t *testing.T) {
// 	r, err := os.Open("testdata/maziusdisplay.cff")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	cffFontFile, err := ParseCFFData(r)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	data := []uint8{0x4a, 0x6f, 0x1d, 0xf7, 0x27, 0xeb, 0x12, 0x96, 0xe7, 0xc8, 0x8e, 0x1d, 0x7d, 0xe7, 0x13, 0xfa, 0xf7, 0x69, 0x58, 0xa, 0x13, 0xfa, 0xf7, 0x12, 0xfc, 0x15, 0x2c, 0x1d}
// 	// data := []uint8{0x51, 0xfb, 0x8e, 0xd0, 0xac, 0xba, 0xac, 0xa2, 0xac, 0xba, 0xac, 0xc7, 0xac, 0xa3, 0xc3, 0xa1, 0xad, 0xa2, 0xac, 0xb0, 0xac, 0xa6, 0xad, 0xaf, 0xad, 0xa5, 0xac, 0xb0, 0xac, 0xce, 0x1, 0xe8, 0xd5, 0xac, 0xac, 0xad, 0xac, 0xac, 0xd8, 0x3, 0xf8, 0x2e, 0x8e, 0xa, 0xfb, 0xd1, 0xfe, 0x7c, 0xf7, 0xd1, 0x6, 0x3e, 0xfa, 0x39, 0x15, 0x6a, 0x49, 0x66, 0xcd, 0x6a, 0xfb, 0x3a, 0xac, 0xcd, 0xb0, 0x4a, 0xac, 0x7, 0xee, 0xfb, 0x15, 0x15, 0x45, 0xcd, 0x69, 0xfb, 0x3a, 0xf3, 0x7, 0xcd, 0x69, 0x15, 0x6a, 0x67, 0xac, 0x6, 0xef, 0x4e, 0x15, 0x6a, 0x49, 0x45, 0x27, 0xac, 0xcd, 0xb0, 0x49, 0xac, 0x7, 0xf7, 0x3a, 0x4f, 0x15, 0x27, 0xfb, 0x3a, 0xad, 0xf7, 0x19, 0xcd, 0x7, 0xac, 0xfb, 0xe, 0x15, 0xfb, 0x5, 0xfb, 0x3a, 0xf7, 0x5, 0xac, 0x3b, 0xef, 0xba, 0x6a, 0x74, 0x69, 0xc3, 0x7, 0xef, 0xfb, 0x41, 0x15, 0xfb, 0x5, 0xfb, 0x3a, 0xf7, 0x5, 0x7, 0xf7, 0x19, 0x6a, 0x15, 0x27, 0x5c, 0xef, 0x6, 0xac, 0x53, 0x15, 0x6a, 0x6b, 0x7, 0x45, 0x5c, 0x5, 0xf1, 0x6a, 0xfb, 0x3a, 0xac, 0x6, 0xd1, 0xba, 0x5, 0x45, 0xac, 0x6, 0xe}
// 	// Toso: test
// 	getSubrsIndex(cffFontFile.globalSubrIndex, cffFontFile.Font[0].subrsIndex, data)

// 	expected := []int{12, 79, 110}
// 	for i, v := range g {
// 		if expected[i] != v {
// 			t.Errorf("g[%d] = %d, want %d", i, v, expected[i])
// 		}
// 	}

// 	expected = []int{7, 8, 56, 62, 89}
// 	for i, v := range l {
// 		if expected[i] != v {
// 			t.Errorf("l[%d] = %d, want %d", i, v, expected[i])
// 		}
// 	}

// }

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
		if ret := cffDictEncodeNumber(td.val); bytes.Compare(ret, td.res) != 0 {
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

// func TestWriteFont(t *testing.T) {
// 	data, err := os.ReadFile("testdata/customfont.cff")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	br := bytes.NewReader(data)
// 	cffFontFile, err := ParseCFFData(br)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	var w bytes.Buffer
// 	err = cffFontFile.WriteCFFData(&w, 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if got, expected := w.Bytes(), data; bytes.Compare(got, expected) != 0 {
// 		maxExpected := 220
// 		maxGot := 220
// 		if l := len(expected); l < maxExpected {
// 			maxExpected = l
// 		}
// 		if l := len(got); l < maxGot {
// 			maxGot = l
// 		}
// 		fmt.Println(expected[:maxExpected])
// 		fmt.Println(got[:maxGot])
// 		t.Error("fonts differ")
// 	}
// }
