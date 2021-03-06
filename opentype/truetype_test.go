package opentype

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateLoca(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "CrimsonPro-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	font, err := Open(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	font.ReadTables()

	bloca, err := font.ReadTableData("loca")
	if err != nil {
		t.Fatal(err)
	}

	var glyphBuffer bytes.Buffer
	font.WriteTable(&glyphBuffer, "glyf")

	var locaBuffer bytes.Buffer
	font.WriteTable(&locaBuffer, "loca")
	if got, want := locaBuffer.Len(), len(bloca); got != want {
		t.Errorf("len(locaBuffer) = %d, want %d", got, want)
	}

	if cmp := bytes.Compare(bloca, locaBuffer.Bytes()); cmp != 0 {
		t.Errorf("compare = %d, want 0 (table loca)", cmp)
	}
}

func TestCompareTables(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "CrimsonPro-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	font, err := Open(f, 0)
	if err != nil {
		t.Fatal(err)
	}

	tables := []string{"hhea", "head", "maxp", "loca", "fpgm", "cvt ", "prep", "glyf"}
	for _, tbl := range tables {
		btbl, err := font.ReadTableData(tbl)
		if err != nil {
			t.Fatal(err)
		}

		err = font.readTable(tbl)
		if err != nil {
			t.Fatal(err)
		}
		var bw bytes.Buffer
		err = font.WriteTable(&bw, tbl)
		if err != nil {
			t.Fatal(err)
		}
		if len(btbl) != bw.Len() {
			t.Errorf("len(bw) = %d, want %d (table %s)", bw.Len(), len(btbl), tbl)
		}

		if cmp := bytes.Compare(btbl, bw.Bytes()); cmp != 0 {
			t.Errorf("compare = %d, want 0 (table %s)", cmp, tbl)
		}

	}
}

func TestWriteFont(t *testing.T) {
	fn := filepath.Join("testdata", "s552.ttf")
	f, err := os.Open(fn)
	if err != nil {
		t.Fatal(err)
	}
	font, err := Open(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	font.ReadTables()

	var buf bytes.Buffer
	err = font.WriteSubset(&buf)
	if err != nil {
		t.Fatal(err)
	}
	w, err := os.Create("dump.ttf")
	if err != nil {
		t.Fatal(err)
	}
	w.Write(buf.Bytes())
	w.Close()
	if got, want := buf.Len(), 276; got != want {
		t.Errorf("len(buf) = %d, want %d", got, want)
	}
}

func TestSubset(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "CrimsonPro-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	font, err := Open(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	font.ReadTables()

	font.Subset([]int{0, 76, 280, 340, 362, 625})

	if is, expected := font.PDFName(), "/FICEFI-CrimsonPro-Regular"; is != expected {
		t.Errorf("font.PDFName() = %s, want %s", is, expected)
	}
	if got, want := font.Ascender(), 918; got != want {
		t.Errorf("font.Ascender() = %d, want %d", got, want)
	}
	if got, want := font.Descender(), -220; got != want {
		t.Errorf("font.Descender() = %d, want %d", got, want)
	}
	if got, want := font.BoundingBox(), "[0 -220 1000 918]"; got != want {
		t.Errorf("font.BoundingBox() = %s, want %s", got, want)
	}
	if got, want := font.Flags(), 4; got != want {
		t.Errorf("font.Flags() = %d, want %d", got, want)
	}
	if got, want := font.ItalicAngle(), 0; got != want {
		t.Errorf("font.ItalicAngle() = %d, want %d", got, want)
	}
	if got, want := font.StemV(), 0; got != want {
		t.Errorf("font.StemV() = %d, want %d", got, want)
	}
	if got, want := font.XHeight(), 425; got != want {
		t.Errorf("font.XHeight() = %d, want %d", got, want)
	}
	if got, want := font.CapHeight(), 587; got != want {
		t.Errorf("font.CapHeight() = %d, want %d", got, want)
	}

	var buf bytes.Buffer
	font.WriteSubset(&buf)
	if got, want := buf.Len(), 5800; got != want {
		t.Errorf("len(buf) = %d, want %d", got, want)
	}
}

func TestWidths(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "CrimsonPro-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	font, err := Open(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	font.ReadTables()
	data := []struct {
		idx int
		wd  int
	}{
		{76, 672},
		{280, 450},
		{340, 269},
	}
	for _, d := range data {
		adv, err := font.GlyphAdvance(d.idx)
		if err != nil {
			t.Error(err)
		}
		if adv != d.wd {
			t.Errorf("font.GlyphAdvance(%d) = %d, want %d", d.idx, adv, d.wd)
		}
	}
}

func TestIndex(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "CrimsonPro-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	font, err := Open(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	font.ReadTables()
	data := []struct {
		r   rune
		idx int
	}{
		{'H', 76},
		{'e', 280},
		{'l', 340},
	}
	for _, d := range data {
		idx, err := font.GetIndex(d.r)
		if err != nil {
			t.Error(err)
		}
		if idx != d.idx {
			t.Errorf("font.GetIndex(%d) = %d, want %d", d.r, idx, d.idx)
		}
	}
}
