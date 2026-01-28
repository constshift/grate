package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pbnjay/grate"
	"github.com/pbnjay/grate/commonxl"
)

var _ = grate.RegisterWithBytes("xlsx", 5, Open, OpenBytes)

// Document contains an Office Open XML document.
type Document struct {
	filename   string
	f          *os.File
	r          *zip.Reader
	primaryDoc string

	// type => id => filename
	rels    map[string]map[string]string
	sheets  []*Sheet
	strings []string
	xfs     []uint16
	fmt     commonxl.Formatter
}

func (d *Document) Close() error {
	d.xfs = d.xfs[:0]
	d.xfs = nil
	d.strings = d.strings[:0]
	d.strings = nil
	d.sheets = d.sheets[:0]
	d.sheets = nil
	if d.f != nil {
		return d.f.Close()
	}
	return nil
}

func Open(filename string) (grate.Source, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	z, err := zip.NewReader(f, info.Size())
	if err != nil {
		return nil, grate.WrapErr(err, grate.ErrNotInFormat)
	}
	return parseDocument(z, filename, f)
}

// OpenBytes opens an XLSX workbook from in-memory data.
func OpenBytes(data []byte) (grate.Source, error) {
	r := bytes.NewReader(data)
	z, err := zip.NewReader(r, int64(len(data)))
	if err != nil {
		return nil, grate.WrapErr(err, grate.ErrNotInFormat)
	}
	return parseDocument(z, "<memory>", nil)
}

// parseDocument is a helper function that parses an XLSX document from a zip reader.
func parseDocument(z *zip.Reader, filename string, f *os.File) (grate.Source, error) {
	d := &Document{
		filename: filename,
		f:        f,
		r:        z,
	}

	d.rels = make(map[string]map[string]string, 4)

	// parse the primary relationships
	dec, c, err := d.openXML("_rels/.rels")
	if err != nil {
		return nil, grate.WrapErr(err, grate.ErrNotInFormat)
	}
	err = d.parseRels(dec, "")
	c.Close()
	if err != nil {
		return nil, grate.WrapErr(err, grate.ErrNotInFormat)
	}
	if d.primaryDoc == "" {
		return nil, errors.New("xlsx: invalid document")
	}

	// parse the secondary relationships to primary doc
	base := filepath.Base(d.primaryDoc)
	sub := strings.TrimSuffix(d.primaryDoc, base)
	relfn := filepath.Join(sub, "_rels", base+".rels")
	dec, c, err = d.openXML(relfn)
	if err != nil {
		return nil, err
	}
	err = d.parseRels(dec, sub)
	c.Close()
	if err != nil {
		return nil, err
	}

	// parse the workbook structure
	dec, c, err = d.openXML(d.primaryDoc)
	if err != nil {
		return nil, err
	}
	err = d.parseWorkbook(dec)
	c.Close()
	if err != nil {
		return nil, err
	}

	styn := d.rels["http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles"]
	for _, sst := range styn {
		// parse the shared string table
		dec, c, err = d.openXML(sst)
		if err != nil {
			return nil, err
		}
		err = d.parseStyles(dec)
		c.Close()
		if err != nil {
			return nil, err
		}
	}

	ssn := d.rels["http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings"]
	for _, sst := range ssn {
		// parse the shared string table
		dec, c, err = d.openXML(sst)
		if err != nil {
			return nil, err
		}
		err = d.parseSharedStrings(dec)
		c.Close()
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}

func (d *Document) openXML(name string) (*xml.Decoder, io.Closer, error) {
	if grate.Debug {
		log.Println("    openXML", name)
	}
	for _, zf := range d.r.File {
		if zf.Name == name {
			zfr, err := zf.Open()
			if err != nil {
				return nil, nil, err
			}
			dec := xml.NewDecoder(zfr)
			return dec, zfr, nil
		}
	}
	return nil, nil, io.EOF
}

func (d *Document) List() ([]string, error) {
	res := make([]string, 0, len(d.sheets))
	for _, s := range d.sheets {
		res = append(res, s.name)
	}
	return res, nil
}

func (d *Document) Get(sheetName string) (grate.Collection, error) {
	for _, s := range d.sheets {
		if s.name == sheetName {
			if s.err == errNotLoaded {
				s.err = s.parseSheet()
			}
			return s.wrapped, s.err
		}
	}
	return nil, errors.New("xlsx: sheet not found")
}
