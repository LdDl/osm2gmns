package osmmicro

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/micro"
	"github.com/LdDl/osm2gmns/csvvalue"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

func ExportToCSV(microNet *micro.Net, outputDir string) error {
	fnameNodes := filepath.Join(outputDir, "micronode.csv")
	fnameLinks := filepath.Join(outputDir, "microlink.csv")

	err := exportNodesToCSV(microNet, fnameNodes)
	if err != nil {
		return errors.Wrap(err, "Can't export micro nodes")
	}

	err = exportLinksToCSV(microNet, fnameLinks)
	if err != nil {
		return errors.Wrap(err, "Can't export micro links")
	}

	return nil
}

func exportNodesToCSV(microNet *micro.Net, fname string) (exportErr error) {
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

	err = writer.Write([]string{
		"id",
		"meso_link_id",
		"lane_id",
		"cell_index",
		"is_upstream_end",
		"is_downstream_end",
		"zone_id",
		"boundary_type",
		"longitude",
		"latitude",
	})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	// Sort node IDs for deterministic output
	sortedNodeIDs := make([]gmns.NodeID, 0, len(microNet.Nodes))
	for id := range microNet.Nodes {
		sortedNodeIDs = append(sortedNodeIDs, id)
	}
	sort.Slice(sortedNodeIDs, func(i, j int) bool {
		return sortedNodeIDs[i].Less(sortedNodeIDs[j])
	})
	for _, nodeID := range sortedNodeIDs {
		node := microNet.Nodes[nodeID]
		err = writer.Write([]string{
			node.ID.String(),
			node.MesoLink().String(),
			fmt.Sprintf("%d", node.LaneID()),
			fmt.Sprintf("%d", node.CellIndex()),
			fmt.Sprintf("%t", node.IsUpstreamEnd()),
			fmt.Sprintf("%t", node.IsDownstreamEnd()),
			csvvalue.NodeID(node.ZoneID()),
			node.BoundaryType().String(),
			fmt.Sprintf("%f", node.Geom()[0]),
			fmt.Sprintf("%f", node.Geom()[1]),
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

func exportLinksToCSV(microNet *micro.Net, fname string) (exportErr error) {
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

	err = writer.Write([]string{
		"id",
		"source_node",
		"target_node",
		"meso_link_id",
		"macro_link_id",
		"macro_node_id",
		"cell_type",
		"lane_id",
		"is_first_movement_cell",
		"movement_composite_type",
		"additional_travel_cost",
		"meso_link_type",
		"control_type",
		"allowed_agent_types",
		"free_speed",
		"capacity",
		"length_meters",
		"geom",
	})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	// Sort link IDs for deterministic output
	sortedLinkIDs := make([]gmns.LinkID, 0, len(microNet.Links))
	for id := range microNet.Links {
		sortedLinkIDs = append(sortedLinkIDs, id)
	}
	sort.Slice(sortedLinkIDs, func(i, j int) bool {
		return sortedLinkIDs[i].Less(sortedLinkIDs[j])
	})
	for _, linkID := range sortedLinkIDs {
		link := microNet.Links[linkID]
		allowedAgentTypes := make([]string, len(link.AllowedAgentTypes()))
		for j, agentType := range link.AllowedAgentTypes() {
			allowedAgentTypes[j] = agentType.String()
		}
		err = writer.Write([]string{
			link.ID.String(),
			link.SourceNode().String(),
			link.TargetNode().String(),
			link.MesoLink().String(),
			link.MacroLink().String(),
			link.MacroNode().String(),
			link.CellType().String(),
			fmt.Sprintf("%d", link.LaneID()),
			fmt.Sprintf("%t", link.IsFirstMovementCell()),
			link.MovementCompositeType().String(),
			fmt.Sprintf("%f", link.AdditionalTravelCost()),
			link.MesoLinkType().String(),
			link.ControlType().String(),
			strings.Join(allowedAgentTypes, ","),
			csvvalue.Float(link.FreeSpeed()),
			csvvalue.Float(link.Capacity()),
			csvvalue.Float(link.LengthMeters()),
			wkt.MarshalString(link.Geom()),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write link")
		}
	}
	writer.Flush()
	err = writer.Error()
	if err != nil {
		return errors.Wrap(err, "Can't flush CSV file")
	}
	return nil
}
