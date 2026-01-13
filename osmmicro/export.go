package osmmicro

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/LdDl/go-gmns/micro"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

func ExportToCSV(microNet *micro.Net, fname string) error {
	fnameParts := strings.Split(fname, ".csv")
	fnameNodes := fmt.Sprintf(fnameParts[0] + "_micro_nodes.csv")
	fnameLinks := fmt.Sprintf(fnameParts[0] + "_micro_links.csv")

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

func exportNodesToCSV(microNet *micro.Net, fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
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

	for i := range microNet.Nodes {
		node := microNet.Nodes[i]
		err = writer.Write([]string{
			fmt.Sprintf("%d", node.ID),
			fmt.Sprintf("%d", node.MesoLink()),
			fmt.Sprintf("%d", node.LaneID()),
			fmt.Sprintf("%d", node.CellIndex()),
			fmt.Sprintf("%t", node.IsUpstreamEnd()),
			fmt.Sprintf("%t", node.IsDownstreamEnd()),
			fmt.Sprintf("%d", node.ZoneID()),
			node.BoundaryType().String(),
			fmt.Sprintf("%f", node.Geom()[0]),
			fmt.Sprintf("%f", node.Geom()[1]),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write node")
		}
	}
	return nil
}

func exportLinksToCSV(microNet *micro.Net, fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
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

	for i := range microNet.Links {
		link := microNet.Links[i]
		allowedAgentTypes := make([]string, len(link.AllowedAgentTypes()))
		for j, agentType := range link.AllowedAgentTypes() {
			allowedAgentTypes[j] = agentType.String()
		}
		err = writer.Write([]string{
			fmt.Sprintf("%d", link.ID),
			fmt.Sprintf("%d", link.SourceNode()),
			fmt.Sprintf("%d", link.TargetNode()),
			fmt.Sprintf("%d", link.MesoLink()),
			fmt.Sprintf("%d", link.MacroLink()),
			fmt.Sprintf("%d", link.MacroNode()),
			link.CellType().String(),
			fmt.Sprintf("%d", link.LaneID()),
			fmt.Sprintf("%t", link.IsFirstMovementCell()),
			link.MovementCompositeType().String(),
			fmt.Sprintf("%f", link.AdditionalTravelCost()),
			link.MesoLinkType().String(),
			link.ControlType().String(),
			strings.Join(allowedAgentTypes, ","),
			fmt.Sprintf("%f", link.FreeSpeed()),
			fmt.Sprintf("%d", link.Capacity()),
			fmt.Sprintf("%f", link.LengthMeters()),
			wkt.MarshalString(link.Geom()),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write link")
		}
	}
	return nil
}
