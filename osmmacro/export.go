package osmmacro

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/csvvalue"
)

func ExportToCSV(net *macro.Net, outputDir string, options ...csvexport.Option) error {
	dataset, err := BuildDataset(net, GEOMETRY_INLINE)
	if err != nil {
		return err
	}
	return csvexport.WriteDatasets(outputDir, []csvexport.Dataset{dataset}, options...)
}

func ExportToCSVWithGeometry(net *macro.Net, outputDir string, mode GeometryMode, options ...csvexport.Option) error {
	dataset, err := BuildDataset(net, mode)
	if err != nil {
		return err
	}
	return csvexport.WriteDatasets(outputDir, []csvexport.Dataset{dataset}, options...)
}

// BuildDataset validates and serializes prepared records without modifying the network.
func BuildDataset(net *macro.Net, mode GeometryMode) (csvexport.Dataset, error) {
	dataset := csvexport.Dataset{Level: "macro"}
	err := net.ValidateDataset()
	if err != nil {
		return dataset, err
	}

	if mode != GEOMETRY_INLINE && mode != GEOMETRY_SHARED {
		return dataset, fmt.Errorf("invalid geometry mode %q", mode)
	}
	shared := mode == GEOMETRY_SHARED
	if shared {
		_, err = collectGeometries(net)
		if err != nil {
			return dataset, err
		}
	}

	nodes := csvexport.Table{
		Name: "node",
		Header: []string{
			"node_id",
			"name",
			"x_coord",
			"y_coord",
			"z_coord",
			"node_type",
			"ctrl_type",
			"zone_id",
			"parent_node_id",
		},
	}
	links := csvexport.Table{
		Name: "link",
		Header: []string{
			"link_id",
			"name",
			"from_node_id",
			"to_node_id",
			"directed",
			"geometry_id",
			"geometry",
			"parent_link_id",
			"dir_flag",
			"length",
			"grade",
			"facility_type",
			"capacity",
			"free_speed",
			"lanes",
			"bike_facility",
			"ped_facility",
			"parking",
			"allowed_uses",
			"toll",
			"jurisdiction",
			"row_width",
		},
	}
	nodeIDs := make([]gmns.NodeID, 0, len(net.Nodes))
	for id := range net.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Slice(nodeIDs, func(first, second int) bool {
		return nodeIDs[first].Less(nodeIDs[second])
	})
	nodeMetadata := make([]map[string]any, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		node := net.Nodes[id]
		nodes.Rows = append(nodes.Rows, []string{
			node.ID.String(),
			node.Name(),
			csvexport.Number(node.Geom()[0]),
			csvexport.Number(node.Geom()[1]),
			csvvalue.Float(node.AltitudeMeters()),
			node.NodeType(),
			node.ControlType().String(),
			csvvalue.NodeID(node.Zone()),
			csvvalue.NodeID(node.ParentNode()),
		})
		nodeMetadata = append(nodeMetadata, map[string]any{
			"node_id":            node.ID,
			"boundary_type":      node.BoundaryType().String(),
			"osm_node_id":        node.OSMNode(),
			"intersection_id":    node.Intersection(),
			"poi_id":             node.POI(),
			"osm_highway":        node.OSMHighway(),
			"activity_type":      node.ActivityType().String(),
			"activity_link_type": node.ActivityLinkType().String(),
		})
	}
	linkIDs := make([]gmns.LinkID, 0, len(net.Links))
	for id := range net.Links {
		linkIDs = append(linkIDs, id)
	}
	sort.Slice(linkIDs, func(first, second int) bool {
		return linkIDs[first].Less(linkIDs[second])
	})
	linkMetadata := make([]map[string]any, 0, len(linkIDs))
	for _, id := range linkIDs {
		link := net.Links[id]
		if len(link.AllowedAgentTypes()) > 0 && link.AllowedUses() == "" {
			return dataset, fmt.Errorf("link %v: prepare allowed_uses before export", id)
		}
		geometryID, hasGeometry := link.GeometryID()
		direction, hasDirection := link.DirFlag()
		geometryCell, shapeCell, directionCell, geometryErr := csvexport.GeometryCells(geometryID, hasGeometry, direction, hasDirection, link.Geom(), link.Directed(), shared)
		if geometryErr != nil {
			return dataset, fmt.Errorf("link %v: %w", id, geometryErr)
		}
		links.Rows = append(links.Rows, []string{
			link.ID.String(),
			link.Name(),
			link.SourceNode().String(),
			link.TargetNode().String(),
			strconv.FormatBool(link.Directed()),
			geometryCell,
			shapeCell,
			csvexport.LinkID(link.ParentLink()),
			directionCell,
			csvvalue.Float(link.LengthMeters()),
			csvvalue.Float(link.Grade()),
			link.FacilityType(),
			csvvalue.Float(link.Capacity()),
			csvvalue.Float(link.FreeSpeed()),
			csvvalue.Int(link.LanesNum()),
			link.BikeFacility(),
			link.PedFacility(),
			link.Parking(),
			link.AllowedUses(),
			csvvalue.Float(link.Toll()),
			link.Jurisdiction(),
			csvvalue.Float(link.RightOfWayWidthMeters()),
		})
		agents := make([]string, 0, len(link.AllowedAgentTypes()))
		for _, agent := range link.AllowedAgentTypes() {
			agents = append(agents, agent.String())
		}
		linkMetadata = append(linkMetadata, map[string]any{
			"geometry_id":         geometryID,
			"source_dir_flag":     direction,
			"link_id":             link.ID,
			"control_type":        link.ControlType().String(),
			"allowed_agent_types": agents,
			"osm_way_id":          link.OSMWay(),
			"source_osm_node_id":  link.SourceOSMNode(),
			"target_osm_node_id":  link.TargetOSMNode(),
			"link_type":           link.LinkType().String(),
			"link_class":          link.LinkClass().String(),
			"is_link":             link.LinkConnectionType().String(),
			"was_bidirectional":   link.WasBidirectional(),
			"max_speed":           link.MaxSpeed(),
		})
	}
	records := net.MovementRecords()
	movements, err := csvexport.Movements(records)
	if err != nil {
		return dataset, err
	}
	movementMetadata := make([]map[string]any, 0, len(records))
	for _, record := range records {
		if len(record.AllowedAgentTypes()) > 0 && record.AllowedUses() == "" {
			return dataset, fmt.Errorf("movement %v: prepare allowed_uses before export", record.ID)
		}
		movementMetadata = append(movementMetadata, map[string]any{
			"movement_id":               record.ID,
			"osm_node_id":               record.OSMNode(),
			"from_osm_node_id":          record.OSMNodeSource(),
			"to_osm_node_id":            record.OSMNodeTarget(),
			"lanes_num":                 record.LanesNum(),
			"incoming_lane_start_index": record.StartIncomeLaneSeqID(),
			"incoming_lane_end_index":   record.EndIncomeLaneSeqID(),
			"outgoing_lane_start_index": record.StartOutcomeLaneSeqID(),
			"outgoing_lane_end_index":   record.EndOutcomeLaneSeqID(),
		})
	}
	dataset.Tables = []csvexport.Table{
		csvexport.Config(net.Config()),
		nodes,
		links,
		csvexport.Geometries(net.GeometryRecords()),
		movements,
		csvexport.Zones(net.ZoneRecords()),
		csvexport.UseDefinitions(net.UseDefinitionRecords()),
		csvexport.UseGroups(net.UseGroupRecords()),
		csvexport.Lanes(net.LaneRecords()),
		csvexport.Segments(net.SegmentRecords()),
		csvexport.SegmentLanes(net.SegmentLaneRecords()),
	}
	dataset.Provenance = map[string]any{
		"format":    "osm2gmns-provenance",
		"level":     "macro",
		"nodes":     nodeMetadata,
		"links":     linkMetadata,
		"movements": movementMetadata,
	}
	return dataset, nil
}
