package csvimport

import (
	"fmt"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/go-gmns/movement"
	"github.com/paulmach/orb"
)

func readRecord(net *macro.Net, name string, row *csvRow) error {
	if name == "config" {
		return readConfig(net, row)
	}
	kind := net.Config().IDType()
	switch name {
	case "node":
		record := readNode(row, kind)
		if row.err != nil {
			return row.err
		}
		if net.Nodes[record.ID] != nil {
			return fmt.Errorf("duplicate node %v", record.ID)
		}
		net.Nodes[record.ID] = record
	case "link":
		record := readLink(row, kind)
		if row.err != nil {
			return row.err
		}
		if net.Links[record.ID] != nil {
			return fmt.Errorf("duplicate link %v", record.ID)
		}
		net.Links[record.ID] = record
	case "lane":
		record := readLane(row, kind)
		if row.err != nil {
			return row.err
		}
		return net.AddLane(record)
	case "segment":
		record := readSegment(row, kind)
		if row.err != nil {
			return row.err
		}
		return net.AddSegment(record)
	case "segment_lane":
		record := readSegmentLane(row, kind)
		if row.err != nil {
			return row.err
		}
		return net.AddSegmentLane(record)
	case "movement":
		record := readMovement(row, kind)
		if row.err != nil {
			return row.err
		}
		return net.AddMovement(record)
	case "geometry":
		record := gmns.NewGeometry(row.geometryID("geometry_id", kind), row.line("geometry"))
		if row.err != nil {
			return row.err
		}
		return net.AddGeometry(record)
	case "zone":
		record := gmns.NewZone(row.nodeID("zone_id", kind),
			gmns.WithZoneName(row.text("name")),
			gmns.WithZoneBoundary(row.boundary("boundary")),
		)
		if row.text("super_zone") != "" {
			gmns.WithSuperZone(row.nodeID("super_zone", kind))(record)
		}
		if row.err != nil {
			return row.err
		}
		return net.AddZone(record)
	case "use_definition":
		record := gmns.NewUseDefinition(row.required("use"), row.number("persons_per_vehicle"), row.number("pce"),
			gmns.WithUseSpecialConditions(row.text("special_conditions")),
			gmns.WithUseDescription(row.text("description")),
		)
		if row.err != nil {
			return row.err
		}
		return net.AddUseDefinition(record)
	case "use_group":
		record := gmns.NewUseGroup(row.required("use_group"), strings.Split(row.required("uses"), ","),
			gmns.WithUseGroupDescription(row.text("description")),
		)
		if row.err != nil {
			return row.err
		}
		return net.AddUseGroup(record)
	default:
		return fmt.Errorf("unsupported table %q", name)
	}
	return nil
}

func readConfig(net *macro.Net, row *csvRow) error {
	for _, field := range []string{"short_length", "long_length"} {
		if row.required(field) != "meter" {
			row.fail(field, fmt.Errorf("only meter is supported"))
		}
	}
	if row.required("speed") != "km/h" {
		row.fail("speed", fmt.Errorf("only km/h is supported"))
	}
	if row.required("crs") != "EPSG:4326" {
		row.fail("crs", fmt.Errorf("only EPSG:4326 is supported"))
	}
	if row.required("geometry_field_format") != "WKT" {
		row.fail("geometry_field_format", fmt.Errorf("only WKT is supported"))
	}
	record := gmns.NewConfig(gmns.IDType(row.required("id_type")),
		gmns.WithDatasetName(row.text("dataset_name")),
		gmns.WithCurrency(row.text("currency")),
	)
	if row.text("version_number") != "" {
		gmns.WithVersionNumber(row.number("version_number"))(record)
	}
	if row.err != nil {
		return row.err
	}
	return net.SetConfig(record)
}

func readNode(row *csvRow, kind gmns.IDType) *macro.Node {
	record := macro.NewNodeFrom(row.nodeID("node_id", kind),
		macro.WithPointGeom(orb.Point{row.number("x_coord"), row.number("y_coord")}),
		macro.WithNodeName(row.text("name")),
		macro.WithNodeType(row.text("node_type")),
		macro.WithNodeControlType(types.ControlType(row.text("ctrl_type"))),
	)
	if row.text("z_coord") != "" {
		macro.WithAltitudeMeters(row.number("z_coord"))(record)
	}
	if row.text("zone_id") != "" {
		macro.WithZoneID(row.nodeID("zone_id", kind))(record)
	}
	if row.text("parent_node_id") != "" {
		macro.WithParentNodeID(row.nodeID("parent_node_id", kind))(record)
	}
	return record
}

func readLink(row *csvRow, kind gmns.IDType) *macro.Link {
	record := macro.NewLinkFrom(row.linkID("link_id", kind), row.nodeID("from_node_id", kind), row.nodeID("to_node_id", kind),
		macro.WithDirected(row.boolean("directed")),
		macro.WithLinkName(row.text("name")),
		macro.WithLineGeom(row.line("geometry")),
		macro.WithFacilityType(row.text("facility_type")),
		macro.WithBikeFacility(row.text("bike_facility")),
		macro.WithPedFacility(row.text("ped_facility")),
		macro.WithParking(row.text("parking")),
		macro.WithAllowedUses(row.text("allowed_uses")),
		macro.WithJurisdiction(row.text("jurisdiction")),
	)
	if row.text("length") != "" {
		macro.WithLengthMeters(row.number("length"))(record)
	}
	if row.text("grade") != "" {
		macro.WithGrade(row.number("grade"))(record)
	}
	if row.text("capacity") != "" {
		macro.WithCapacity(row.number("capacity"))(record)
	}
	if row.text("free_speed") != "" {
		macro.WithFreeSpeed(row.number("free_speed"))(record)
	}
	if row.text("toll") != "" {
		macro.WithToll(row.number("toll"))(record)
	}
	if row.text("row_width") != "" {
		macro.WithRightOfWayWidthMeters(row.number("row_width"))(record)
	}
	if row.text("lanes") != "" {
		macro.WithLanesNum(row.integer("lanes"))(record)
	}
	if row.text("geometry_id") != "" {
		macro.WithGeometryID(row.geometryID("geometry_id", kind))(record)
	}
	if row.text("parent_link_id") != "" {
		macro.WithParentLinkID(row.linkID("parent_link_id", kind))(record)
	}
	if row.text("dir_flag") != "" {
		direction := row.integer("dir_flag")
		switch direction {
		case -1, 1:
			macro.WithDirFlag(gmns.DirectionFlag(direction))(record)
		case 0:
			if record.Directed() {
				row.fail("dir_flag", fmt.Errorf("0 requires an undirected link"))
			}
		default:
			row.fail("dir_flag", fmt.Errorf("expected -1, 0 or 1"))
		}
	}
	_, shared := record.GeometryID()
	direction, present := record.DirFlag()
	if !shared && present && direction == gmns.DIR_FLAG_REVERSE {
		record.Geom().Reverse()
		macro.WithDirFlag(gmns.DIR_FLAG_FORWARD)(record)
	}
	return record
}

func readMovement(row *csvRow, kind gmns.IDType) *movement.Movement {
	record := movement.NewMovement(row.movementID("mvmt_id", kind), row.nodeID("node_id", kind),
		row.linkID("ib_link_id", kind), row.linkID("ob_link_id", kind),
		movement.MovementCompositeType(row.text("mvmt_code")), movement.MovementType(row.text("type")),
		movement.WithName(row.text("name")),
		movement.WithControlType(types.ControlType(row.text("ctrl_type"))),
		movement.WithAllowedUses(row.text("allowed_uses")),
		movement.WithGeom(row.line("geometry")),
	)
	if row.text("start_ib_lane") != "" {
		movement.WithIncomeLaneStart(row.integer("start_ib_lane"))(record)
	}
	if row.text("end_ib_lane") != "" {
		movement.WithIncomeLaneEnd(row.integer("end_ib_lane"))(record)
	}
	if row.text("start_ob_lane") != "" {
		movement.WithOutcomeLaneStart(row.integer("start_ob_lane"))(record)
	}
	if row.text("end_ob_lane") != "" {
		movement.WithOutcomeLaneEnd(row.integer("end_ob_lane"))(record)
	}
	if row.text("penalty") != "" {
		movement.WithPenalty(row.number("penalty"))(record)
	}
	if row.text("capacity") != "" {
		movement.WithCapacity(row.number("capacity"))(record)
	}
	return record
}
