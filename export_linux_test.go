package osm2gmns_test

import (
	"os"
	"syscall"
	"testing"

	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/stretchr/testify/assert"
)

func TestCSVFlushErrors(t *testing.T) {
	_, err := os.Stat("/dev/full")
	if err != nil {
		t.Skip("requires /dev/full to simulate a full disk")
	}
	table := csvexport.Table{
		Name:   "node",
		Header: []string{"node_id", "x_coord", "y_coord"},
		Rows: [][]string{
			{"1", "0", "0"},
		},
	}
	err = csvexport.WriteTable("/dev/full", table)
	assert.ErrorIs(t, err, syscall.ENOSPC)
}
