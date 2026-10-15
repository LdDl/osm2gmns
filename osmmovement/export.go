package osmmovement

import (
	"fmt"

	"github.com/LdDl/go-gmns/movement"
	"github.com/LdDl/osm2gmns/csvexport"
)

// ExportToCSV writes a standalone standard movement table.
// Export a network dataset when configuration and local reference validation are needed.
func ExportToCSV(storage movement.MovementsStorage, filename string, options ...csvexport.Option) error {
	records := make([]*movement.Movement, 0, len(storage))
	for id, record := range storage {
		if record == nil || record.ID != id {
			return fmt.Errorf("movement %v: invalid storage entry", id)
		}
		records = append(records, record)
	}
	table, err := csvexport.Movements(records)
	if err != nil {
		return err
	}
	return csvexport.WriteTable(filename, table, options...)
}
