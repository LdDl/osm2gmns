package csvexport

import (
	"encoding/csv"
	"fmt"
	"os"
)

// Table contains serialized records, without network generation state.
type Table struct {
	Name   string
	Header []string
	Rows   [][]string
}

// Dataset groups the tables of one network level and separate application metadata.
type Dataset struct {
	Level      string
	Tables     []Table
	Provenance map[string]any
}

// WriteTable writes a CSV table and reports buffered write errors.
func WriteTable(filename string, table Table, configure ...Option) error {
	settings, err := resolveOptions(configure)
	if err != nil {
		return err
	}
	err = validateTable(table)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	writer.Comma = settings.delimiter
	err = writer.Write(table.Header)
	if err == nil {
		for _, row := range table.Rows {
			err = writer.Write(row)
			if err != nil {
				break
			}
		}
	}
	writer.Flush()
	flushErr := writer.Error()
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}

func validateTable(table Table) error {
	for _, row := range table.Rows {
		if len(row) != len(table.Header) {
			return fmt.Errorf("%s: row has %d fields, expected %d", table.Name, len(row), len(table.Header))
		}
		for _, value := range row {
			if value == "NaN" {
				return fmt.Errorf("%s: literal NaN conflicts with the schema's missing-value marker", table.Name)
			}
		}
	}
	return nil
}
