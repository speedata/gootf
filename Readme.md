# OpenType and TrueType font reader, writer and subsetter for Go

This code allows you to read and write OpenType (CFF) and TrueType fonts. When writing the fonts you can chose to subset the fonts so that only the font code necessary for the chosen glyphs are put into the file.

Used in https://github.com/speedata/boxesandglue for PDF embedding.

## Usage

```go
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/speedata/gootf/opentype"
)

func dothings() error {
	r, err := os.Open("DejaVuSerif.ttf")
	if err != nil {
		return err
	}
	// 0 is the font index of the subfont
	tt, err := opentype.Open(r, 0)
	if err != nil {
		return err
	}
	tt.ReadTables()
	fmt.Println(tt.FontName) // DejaVuSerif

	// Subset font for code points 1,2,3 and 4.
	if err = tt.Subset([]int{1, 2, 3, 4}); err != nil {
		return err
	}
	// some “random” string for PDF
	fmt.Println(tt.SubsetID)

	var fontchunk bytes.Buffer
	tt.WriteSubset(&fontchunk)
	fmt.Println(len(fontchunk.Bytes())) // 2260

	return r.Close()
}

func main() {
	if err := dothings(); err != nil {
		log.Fatal(err)
	}
}
```


License: 3-Clause BSD License<br>
Status: Supported.<br>
Contact: <gundlach@speedata.de>

