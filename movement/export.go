package movement

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

type MovementsStorage map[MovementID]*Movement

func NewMovementsStorage() map[MovementID]*Movement {
	return make(map[MovementID]*Movement)
}

func (mvmtStorage MovementsStorage) ExportToCSV(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "node_id", "osm_node_id", "name", "in_link_id", "in_lane_start", "in_lane_end", "out_link_id", "out_lane_start", "out_lane_end", "lanes_num", "from_osm_node_id", "to_osm_node_id", "type", "penalty", "capacity", "control_type", "movement_composite_type", "volume", "free_speed", "allowed_agent_types", "geom"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for k := range mvmtStorage {
		mvmt := mvmtStorage[k]
		allowedAgentTypes := make([]string, len(mvmt.allowedAgentTypes))
		for i, agentType := range mvmt.allowedAgentTypes {
			allowedAgentTypes[i] = agentType.String()
		}
		err = writer.Write([]string{
			fmt.Sprintf("%d", mvmt.ID),
			fmt.Sprintf("%d", mvmt.MacroNodeID),
			fmt.Sprintf("%d", mvmt.osmNodeID),
			mvmt.name,
			fmt.Sprintf("%d", mvmt.IncomeMacroLinkID),
			fmt.Sprintf("%d", mvmt.incomeLaneStart),
			fmt.Sprintf("%d", mvmt.incomeLaneEnd),
			fmt.Sprintf("%d", mvmt.OutcomeMacroLinkID),
			fmt.Sprintf("%d", mvmt.outcomeLaneStart),
			fmt.Sprintf("%d", mvmt.outcomeLaneEnd),
			fmt.Sprintf("%d", mvmt.lanesNum),
			fmt.Sprintf("%d", mvmt.fromOsmNodeID),
			fmt.Sprintf("%d", mvmt.toOsmNodeID),
			mvmt.MType.String(),
			fmt.Sprintf("%d", -1),
			fmt.Sprintf("%d", -1),
			mvmt.controlType.String(),
			mvmt.MTextID.String(),
			fmt.Sprintf("%d", -1),
			fmt.Sprintf("%d", -1),
			strings.Join(allowedAgentTypes, ","),
			wkt.MarshalString(mvmt.Geom),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write node")
		}
	}
	return nil
}
