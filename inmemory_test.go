package grate_test

import (
	"os"
	"testing"

	"github.com/constshift/grate"
	_ "github.com/constshift/grate/simple"
	_ "github.com/constshift/grate/xls"
	_ "github.com/constshift/grate/xlsx"
)

func TestOpenBytes(t *testing.T) {
	testFiles := []string{
		"testdata/basic.tsv",
		"testdata/basic.xls",
		"testdata/basic.xlsx",
	}

	for _, filename := range testFiles {
		t.Run(filename, func(t *testing.T) {
			// Read file into memory
			data, err := os.ReadFile(filename)
			if err != nil {
				if os.IsNotExist(err) {
					t.Skipf("Test file %s does not exist", filename)
					return
				}
				t.Fatalf("Failed to read test file %s: %v", filename, err)
			}

			// Open from memory
			wb, err := grate.OpenBytes(data)
			if err != nil {
				t.Fatalf("Failed to open %s from bytes: %v", filename, err)
			}
			defer wb.Close()

			// Get sheet list
			sheets, err := wb.List()
			if err != nil {
				t.Fatalf("Failed to list sheets from %s: %v", filename, err)
			}

			if len(sheets) == 0 {
				t.Fatalf("Expected at least one sheet in %s", filename)
			}

			// Open first sheet and read some data
			sheet, err := wb.Get(sheets[0])
			if err != nil {
				t.Fatalf("Failed to get sheet from %s: %v", filename, err)
			}

			rowCount := 0
			for sheet.Next() {
				row := sheet.Strings()
				if len(row) == 0 {
					t.Errorf("Expected non-empty row in %s", filename)
				}
				rowCount++
			}

			if rowCount == 0 {
				t.Errorf("Expected at least one row in %s", filename)
			}

			if sheet.Err() != nil {
				t.Errorf("Error iterating sheet from %s: %v", filename, sheet.Err())
			}
		})
	}
}
