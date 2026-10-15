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
	"github.com/LdDl/osm2gmns/csvvalue"
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

func exportNodesToCSV(mesoNet *meso.Net, fname string) (exportErr error) {
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
		return sortedNodeIDs[i].Less(sortedNodeIDs[j])
	})
	for _, nodeID := range sortedNodeIDs {
		node := mesoNet.Nodes[nodeID]
		err = writer.Write([]string{
			node.ID.String(),
			csvvalue.NodeID(node.MacroZone()),
			node.MacroNode().String(),
			node.MacroLink().String(),
			node.ActivityLinkType().String(),
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

func exportLinksToCSV(mesoNet *meso.Net, fname string) (exportErr error) {
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
		return sortedLinkIDs[i].Less(sortedLinkIDs[j])
	})
	for _, linkID := range sortedLinkIDs {
		link := mesoNet.Links[linkID]
		allowedAgentTypes := make([]string, len(link.AllowedAgentTypes()))
		for i, agentType := range link.AllowedAgentTypes() {
			allowedAgentTypes[i] = agentType.String()
		}
		err = writer.Write([]string{
			link.ID.String(),
			link.SourceNode().String(),
			link.TargetNode().String(),
			link.MacroNode().String(),
			link.MacroLink().String(),
			link.LinkType().String(),
			link.ControlType().String(),
			link.Movement().String(),
			link.MvmtTextID().String(),
			strings.Join(allowedAgentTypes, ","),
			csvvalue.Int(link.LanesNum()),
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
