package macro

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

func (net *Net) ExportToCSV(fname string) error {
	fnameParts := strings.Split(fname, ".csv")
	fnameNodes := fmt.Sprintf(fnameParts[0] + "_macro_nodes.csv")
	fnameLinks := fmt.Sprintf(fnameParts[0] + "_macro_links.csv")
	// fnameMovement := fmt.Sprintf(fnameParts[0] + "_movement.csv")

	err := net.exportNodesToCSV(fnameNodes)
	if err != nil {
		return errors.Wrap(err, "Can't export nodes")
	}

	err = net.exportLinksToCSV(fnameLinks)
	if err != nil {
		return errors.Wrap(err, "Can't export links")
	}

	// err = net.exportMovementToCSV(fnameMovement)
	// if err != nil {
	// return errors.Wrap(err, "Can't export movement")
	// }
	return nil
}

func (net *Net) exportNodesToCSV(fname string) error {
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

	for i := range net.Nodes {
		node := net.Nodes[i]
		err = writer.Write([]string{
			fmt.Sprintf("%d", node.ID),
			fmt.Sprintf("%d", node.osmNodeID),
			node.controlType.String(),
			node.boundaryType.String(),
			node.activityType.String(),
			node.activityLinkType.String(),
			fmt.Sprintf("%d", node.zoneID),
			fmt.Sprintf("%d", node.intersectionID),
			fmt.Sprintf("%d", node.poiID),
			node.osmHighway,
			node.name,
			fmt.Sprintf("%f", node.geom[0]),
			fmt.Sprintf("%f", node.geom[1]),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write node")
		}
	}
	return nil
}

func (net *Net) exportLinksToCSV(fname string) error {
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

	for i := range net.Links {
		link := net.Links[i]
		allowedAgentTypes := make([]string, len(link.allowedAgentTypes))
		for i, agentType := range link.allowedAgentTypes {
			allowedAgentTypes[i] = agentType.String()
		}
		err = writer.Write([]string{
			fmt.Sprintf("%d", link.ID),
			fmt.Sprintf("%d", link.sourceNodeID),
			fmt.Sprintf("%d", link.targetNodeID),
			fmt.Sprintf("%d", link.osmWayID),
			fmt.Sprintf("%d", link.sourceOsmNodeID),
			fmt.Sprintf("%d", link.targetOsmNodeID),
			link.linkClass.String(),
			link.linkConnectionType.String(),
			link.linkType.String(),
			link.controlType.String(),
			strings.Join(allowedAgentTypes, ","),
			fmt.Sprintf("%t", link.wasBidirectional),
			fmt.Sprintf("%d", link.lanesNum),
			fmt.Sprintf("%f", link.maxSpeed),
			fmt.Sprintf("%f", link.freeSpeed),
			fmt.Sprintf("%d", link.capacity),
			fmt.Sprintf("%f", link.lengthMeters),
			link.name,
			wkt.MarshalString(link.geom),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write link")
		}
	}
	return nil
}
