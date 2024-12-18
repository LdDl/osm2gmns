package osmmacro

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/LdDl/go-gmns/macro"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

func ExportToCSV(macroNet *macro.Net, fname string) error {
	fnameParts := strings.Split(fname, ".csv")
	fnameNodes := fmt.Sprintf(fnameParts[0] + "_macro_nodes.csv")
	fnameLinks := fmt.Sprintf(fnameParts[0] + "_macro_links.csv")

	err := exportNodesToCSV(macroNet, fnameNodes)
	if err != nil {
		return errors.Wrap(err, "Can't export nodes")
	}

	err = exportLinksToCSV(macroNet, fnameLinks)
	if err != nil {
		return errors.Wrap(err, "Can't export links")
	}
	return nil
}

func exportNodesToCSV(macroNet *macro.Net, fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "osm_node_id", "control_type", "boundary_type", "activity_type", "activity_link_type", "zone_id", "intersection_id", "poi_id", "osm_highway", "name", "longitude", "latitude"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for i := range macroNet.Nodes {
		node := macroNet.Nodes[i]
		err = writer.Write([]string{
			fmt.Sprintf("%d", node.ID),
			fmt.Sprintf("%d", node.OSMNode()),
			node.ControlType().String(),
			node.BoundaryType().String(),
			node.ActivityType().String(),
			node.ActivityLinkType().String(),
			fmt.Sprintf("%d", node.Zone()),
			fmt.Sprintf("%d", node.Intersection()),
			fmt.Sprintf("%d", node.POI()),
			node.OSMHighway(),
			node.Name(),
			fmt.Sprintf("%f", node.Geom()[0]),
			fmt.Sprintf("%f", node.Geom()[1]),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write node")
		}
	}
	return nil
}

func exportLinksToCSV(macroNet *macro.Net, fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "source_node", "target_node", "osm_way_id", "source_osm_node_id", "target_osm_node_id", "link_class", "is_link", "link_type", "control_type", "allowed_agent_types", "was_bidirectional", "lanes", "max_speed", "free_speed", "capacity", "length_meters", "name", "geom"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for i := range macroNet.Links {
		link := macroNet.Links[i]
		allowedAgentTypes := make([]string, len(link.AllowedAgentTypes()))
		for i, agentType := range link.AllowedAgentTypes() {
			allowedAgentTypes[i] = agentType.String()
		}
		err = writer.Write([]string{
			fmt.Sprintf("%d", link.ID),
			fmt.Sprintf("%d", link.SourceNode()),
			fmt.Sprintf("%d", link.TargetNode()),
			fmt.Sprintf("%d", link.OSMWay()),
			fmt.Sprintf("%d", link.SourceOSMNode()),
			fmt.Sprintf("%d", link.TargetOSMNode()),
			link.LinkClass().String(),
			link.LinkConnectionType().String(),
			link.LinkType().String(),
			link.ControlType().String(),
			strings.Join(allowedAgentTypes, ","),
			fmt.Sprintf("%t", link.WasBidirectional()),
			fmt.Sprintf("%d", link.LanesNum()),
			fmt.Sprintf("%f", link.MaxSpeed()),
			fmt.Sprintf("%f", link.FreeSpeed()),
			fmt.Sprintf("%d", link.Capacity()),
			fmt.Sprintf("%f", link.LengthMeters()),
			link.Name(),
			wkt.MarshalString(link.Geom()),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write link")
		}
	}
	return nil
}
