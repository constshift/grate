// Example demonstrating in-memory file processing with grate
package main

import (
	"fmt"
	"os"

	"github.com/constshift/grate"
	_ "github.com/constshift/grate/simple"
	_ "github.com/constshift/grate/xls"
	_ "github.com/constshift/grate/xlsx"
)

// Example 1: Opening from byte slice
func fromBytes() error {
	// Read file into memory
	data, err := os.ReadFile("../testdata/basic.xlsx")

	// Open from memory
	wb, err := grate.OpenBytes(data)
	if err != nil {
		return err
	}
	defer wb.Close()

	sheets, _ := wb.List()
	sheet, _ := wb.Get(sheets[0])

	fmt.Println("Example 1: From Bytes")
	for sheet.Next() {
		fmt.Println(sheet.Strings())
	}
	return nil
}

func main() {
	if err := fromBytes(); err != nil {
		fmt.Fprintf(os.Stderr, "Error in fromBytes: %v\n", err)
	}
}
