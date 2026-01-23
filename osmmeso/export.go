package osmmeso

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/meso"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

func ExportToCSV(mesoNet *meso.Net, outputDir string) error {
	fnameNodes := filepath.Join(outputDir, "mesonode.csv")
	fnameLinks := filepath.Join(outputDir, "mesolink.csv")

	err := exportNodesToCSV(mesoNet, fnameNodes)
	if err != nil {
		return errors.Wrap(err, "Can't export nodes")
	}

	err = exportLinksToCSV(mesoNet, fnameLinks)
	if err != nil {
		return errors.Wrap(err, "Can't export links")
	}

	return nil
}

func exportNodesToCSV(mesoNet *meso.Net, fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "zone_id", "macro_node_id", "macro_link_id", "activity_link_type", "boundary_type", "longitude", "latitude"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	// Sort node IDs for deterministic output
	sortedNodeIDs := make([]gmns.NodeID, 0, len(mesoNet.Nodes))
	for id := range mesoNet.Nodes {
		sortedNodeIDs = append(sortedNodeIDs, id)
	}
	sort.Slice(sortedNodeIDs, func(i, j int) bool {
		return sortedNodeIDs[i] < sortedNodeIDs[j]
	})
	for _, nodeID := range sortedNodeIDs {
		node := mesoNet.Nodes[nodeID]
		err = writer.Write([]string{
			fmt.Sprintf("%d", node.ID),
			fmt.Sprintf("%d", node.MacroZone()),
			fmt.Sprintf("%d", node.MacroNode()),
			fmt.Sprintf("%d", node.MacroLink()),
			node.ActivityLinkType().String(),
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

func exportLinksToCSV(mesoNet *meso.Net, fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "source_node", "target_node", "macro_node_id", "macro_link_id", "link_type", "control_type", "movement_id", "movement_composite_type", "allowed_agent_types", "lanes", "free_speed", "capacity", "length_meters", "geom"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	// Sort link IDs for deterministic output
	sortedLinkIDs := make([]gmns.LinkID, 0, len(mesoNet.Links))
	for id := range mesoNet.Links {
		sortedLinkIDs = append(sortedLinkIDs, id)
	}
	sort.Slice(sortedLinkIDs, func(i, j int) bool {
		return sortedLinkIDs[i] < sortedLinkIDs[j]
	})
	for _, linkID := range sortedLinkIDs {
		link := mesoNet.Links[linkID]
		allowedAgentTypes := make([]string, len(link.AllowedAgentTypes()))
		for i, agentType := range link.AllowedAgentTypes() {
			allowedAgentTypes[i] = agentType.String()
		}
		err = writer.Write([]string{
			fmt.Sprintf("%d", link.ID),
			fmt.Sprintf("%d", link.SourceNode()),
			fmt.Sprintf("%d", link.TargetNode()),
			fmt.Sprintf("%d", link.MacroNode()),
			fmt.Sprintf("%d", link.MacroLink()),
			link.LinkType().String(),
			link.ControlType().String(),
			fmt.Sprintf("%d", link.Movement()),
			link.MvmtTextID().String(),
			strings.Join(allowedAgentTypes, ","),
			fmt.Sprintf("%d", link.LanesNum()),
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
