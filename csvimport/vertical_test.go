package csvimport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThreeDimensionalWKTIsRejectedWithoutFlattening(t *testing.T) {
	for _, text := range []string{"LINESTRING Z (37 55 0,37.1 55.1 10)", "LINESTRING(37 55 0,37.1 55.1 10)", "LINESTRING M (37 55 0,37.1 55.1 10)"} {
		row := &csvRow{values: map[string]string{"geometry": text}}
		row.line("geometry")
		assert.Error(t, row.err, text)
	}
}
