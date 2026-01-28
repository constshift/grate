package simple

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/pbnjay/grate"
)

var _ = grate.RegisterWithBytes("tsv", 10, OpenTSV, OpenTSVBytes)

// OpenTSV defines a Source's instantiation function.
// It should return ErrNotInFormat immediately if filename is not of the correct file type.
func OpenTSV(filename string) (grate.Source, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseTSV(f, filename)
}

// OpenTSVBytes opens TSV data from an in-memory byte slice.
func OpenTSVBytes(data []byte) (grate.Source, error) {
	r := bytes.NewReader(data)
	return parseTSV(r, "<memory>")
}

// parseTSV is a helper function that parses TSV data from an io.Reader.
func parseTSV(r io.Reader, filename string) (grate.Source, error) {
	t := &simpleFile{
		filename: filename,
		iterRow:  -1,
	}

	s := bufio.NewScanner(r)
	total := 0
	ncols := make(map[int]int)
	for s.Scan() {
		row := strings.Split(s.Text(), "\t")
		ncols[len(row)]++
		total++
		t.rows = append(t.rows, row)
	}
	if s.Err() != nil {
		// this can only be read errors, not format
		return nil, s.Err()
	}

	// kinda arbitrary metrics for detecting TSV
	looksGood := 0
	for c, n := range ncols {
		if c <= 1 {
			continue
		}
		if n > 10 && float64(n)/float64(total) > 0.8 {
			// more than 80% of rows have the same number of columns, we're good
			looksGood = 2
		} else if n > 25 && looksGood == 0 {
			looksGood = 1
		}
	}
	if looksGood == 1 {
		return t, grate.ErrNotInFormat
	}

	return t, nil
}
