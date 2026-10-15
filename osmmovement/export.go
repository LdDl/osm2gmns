package osmmovement

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/movement"
	"github.com/LdDl/osm2gmns/csvvalue"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

func ExportToCSV(mvmtStorage movement.MovementsStorage, fname string) (exportErr error) {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer func() {
		closeErr := file.Close()
		if exportErr == nil && closeErr != nil {
			exportErr = errors.Wrap(closeErr, "Can't close CSV file")
		}
	}()

	writer := csv.NewWriter(file)
	writer.Comma = ';'

	err = writer.Write([]string{"id", "node_id", "osm_node_id", "name", "in_link_id", "in_lane_start", "in_lane_end", "out_link_id", "out_lane_start", "out_lane_end", "lanes_num", "from_osm_node_id", "to_osm_node_id", "type", "penalty", "capacity", "control_type", "movement_composite_type", "volume", "free_speed", "allowed_agent_types", "geom"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	// Sort movement IDs for deterministic output
	sortedMvmtIDs := make([]gmns.MovementID, 0, len(mvmtStorage))
	for id := range mvmtStorage {
		sortedMvmtIDs = append(sortedMvmtIDs, id)
	}
	sort.Slice(sortedMvmtIDs, func(i, j int) bool {
		return sortedMvmtIDs[i].Less(sortedMvmtIDs[j])
	})
	for _, mvmtID := range sortedMvmtIDs {
		mvmt := mvmtStorage[mvmtID]
		allowedAgentTypes := make([]string, len(mvmt.AllowedAgentTypes()))
		for i, agentType := range mvmt.AllowedAgentTypes() {
			allowedAgentTypes[i] = agentType.String()
		}
		err = writer.Write([]string{
			mvmt.ID.String(),
			mvmt.Node().String(),
			fmt.Sprintf("%d", mvmt.OSMNode()),
			mvmt.Name(),
			mvmt.IncomeLink().String(),
			csvvalue.Int(mvmt.IncomeLaneStart()),
			csvvalue.Int(mvmt.IncomeLaneEnd()),
			mvmt.OutcomeLink().String(),
			csvvalue.Int(mvmt.OutcomeLaneStart()),
			csvvalue.Int(mvmt.OutcomeLaneEnd()),
			fmt.Sprintf("%d", mvmt.LanesNum()),
			fmt.Sprintf("%d", mvmt.OSMNodeSource()),
			fmt.Sprintf("%d", mvmt.OSMNodeTarget()),
			mvmt.Type().String(),
			csvvalue.Float(mvmt.Penalty()),
			csvvalue.Float(mvmt.Capacity()),
			mvmt.ControlType().String(),
			mvmt.MvmtTextID().String(),
			"",
			"",
			strings.Join(allowedAgentTypes, ","),
			wkt.MarshalString(mvmt.Geom()),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write node")
		}
	}
	writer.Flush()
	err = writer.Error()
	if err != nil {
		return errors.Wrap(err, "Can't flush CSV file")
	}
	return nil
}
