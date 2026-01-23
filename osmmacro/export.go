package osmmacro

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

func ExportToCSV(macroNet *macro.Net, outputDir string) error {
	fnameNodes := filepath.Join(outputDir, "node.csv")
	fnameLinks := filepath.Join(outputDir, "link.csv")

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

	// Sort node IDs for deterministic output
	sortedNodeIDs := make([]gmns.NodeID, 0, len(macroNet.Nodes))
	for id := range macroNet.Nodes {
		sortedNodeIDs = append(sortedNodeIDs, id)
	}
	sort.Slice(sortedNodeIDs, func(i, j int) bool {
		return sortedNodeIDs[i] < sortedNodeIDs[j]
	})
	for _, nodeID := range sortedNodeIDs {
		node := macroNet.Nodes[nodeID]
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

	// Sort link IDs for deterministic output
	sortedLinkIDs := make([]gmns.LinkID, 0, len(macroNet.Links))
	for id := range macroNet.Links {
		sortedLinkIDs = append(sortedLinkIDs, id)
	}
	sort.Slice(sortedLinkIDs, func(i, j int) bool {
		return sortedLinkIDs[i] < sortedLinkIDs[j]
	})
	for _, linkID := range sortedLinkIDs {
		link := macroNet.Links[linkID]
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
