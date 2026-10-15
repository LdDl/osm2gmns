package csvexport

import (
	"fmt"
	"sort"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/movement"
	"github.com/LdDl/osm2gmns/csvvalue"
)

func Movements(records []*movement.Movement) (Table, error) {
	table := Table{
		Name: "movement",
		Header: []string{
			"mvmt_id",
			"node_id",
			"name",
			"ib_link_id",
			"start_ib_lane",
			"end_ib_lane",
			"ob_link_id",
			"start_ob_lane",
			"end_ob_lane",
			"type",
			"penalty",
			"capacity",
			"ctrl_type",
			"mvmt_code",
			"allowed_uses",
			"geometry",
		},
	}
	seen := make(map[gmns.MovementID]bool)
	records = append([]*movement.Movement(nil), records...)
	for _, record := range records {
		err := record.Validate()
		if err != nil {
			return table, err
		}
		if seen[record.ID] {
			return table, fmt.Errorf("duplicate movement %v", record.ID)
		}
		seen[record.ID] = true
		if len(record.AllowedAgentTypes()) > 0 && record.AllowedUses() == "" {
			return table, fmt.Errorf("movement %v: prepare allowed_uses before export", record.ID)
		}
		if !ValidMovementCode(record.MvmtTextID().String()) {
			return table, fmt.Errorf("movement %v: unsupported standard mvmt_code %q", record.ID, record.MvmtTextID())
		}
	}
	sort.Slice(records, func(first, second int) bool {
		return records[first].ID.Less(records[second].ID)
	})
	for _, record := range records {
		table.Rows = append(table.Rows, []string{
			record.ID.String(),
			record.Node().String(),
			record.Name(),
			record.IncomeLink().String(),
			csvvalue.Int(record.IncomeLaneStart()),
			csvvalue.Int(record.IncomeLaneEnd()),
			record.OutcomeLink().String(),
			csvvalue.Int(record.OutcomeLaneStart()),
			csvvalue.Int(record.OutcomeLaneEnd()),
			record.Type().String(),
			csvvalue.Float(record.Penalty()),
			csvvalue.Float(record.Capacity()),
			record.ControlType().String(),
			record.MvmtTextID().String(),
			record.AllowedUses(),
			Line(record.Geom()),
		})
	}
	return table, nil
}
