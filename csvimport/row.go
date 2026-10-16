package csvimport

import (
	"fmt"
	"strconv"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
)

// csvRow accumulates the first field error while constructing one record.
type csvRow struct {
	values map[string]string
	err    error
}

func (row *csvRow) fail(field string, err error) {
	if row.err == nil && err != nil {
		row.err = fmt.Errorf("%s: %w", field, err)
	}
}

func (row *csvRow) text(field string) string {
	value := row.values[field]
	if value == "NaN" {
		return ""
	}
	return value
}

func (row *csvRow) required(field string) string {
	value := row.text(field)
	if value == "" {
		row.fail(field, fmt.Errorf("value is required"))
	}
	return value
}

func (row *csvRow) number(field string) float64 {
	value, err := strconv.ParseFloat(row.required(field), 64)
	row.fail(field, err)
	if !gmns.Finite(value) {
		row.fail(field, fmt.Errorf("value must be finite"))
	}
	return value
}

func (row *csvRow) integer(field string) int {
	value, err := strconv.Atoi(row.required(field))
	row.fail(field, err)
	return value
}

func (row *csvRow) boolean(field string) bool {
	value := row.required(field)
	switch value {
	case "true", "True", "TRUE", "1":
		return true
	case "false", "False", "FALSE", "0":
		return false
	default:
		row.fail(field, fmt.Errorf("invalid boolean %q", value))
		return false
	}
}

func (row *csvRow) nodeID(field string, kind gmns.IDType) gmns.NodeID {
	value, err := gmns.ParseNodeID(row.required(field), kind)
	row.fail(field, err)
	return value
}

func (row *csvRow) linkID(field string, kind gmns.IDType) gmns.LinkID {
	value, err := gmns.ParseLinkID(row.required(field), kind)
	row.fail(field, err)
	return value
}

func (row *csvRow) geometryID(field string, kind gmns.IDType) gmns.GeometryID {
	value, err := gmns.ParseGeometryID(row.required(field), kind)
	row.fail(field, err)
	return value
}

func (row *csvRow) movementID(field string, kind gmns.IDType) gmns.MovementID {
	value, err := gmns.ParseMovementID(row.required(field), kind)
	row.fail(field, err)
	return value
}

func (row *csvRow) line(field string) orb.LineString {
	text := row.text(field)
	if text == "" {
		return nil
	}
	value, err := wkt.Unmarshal(text)
	row.fail(field, err)
	line, ok := value.(orb.LineString)
	if !ok {
		row.fail(field, fmt.Errorf("expected LINESTRING"))
	}
	return line
}

func (row *csvRow) boundary(field string) orb.MultiPolygon {
	text := row.text(field)
	if text == "" {
		return nil
	}
	value, err := wkt.Unmarshal(text)
	row.fail(field, err)
	switch shape := value.(type) {
	case orb.Polygon:
		return orb.MultiPolygon{shape}
	case orb.MultiPolygon:
		return shape
	default:
		row.fail(field, fmt.Errorf("expected POLYGON or MULTIPOLYGON"))
		return nil
	}
}

func (row *csvRow) laneID(field string, kind gmns.IDType) gmns.LaneID {
	value, err := gmns.ParseLaneID(row.required(field), kind)
	row.fail(field, err)
	return value
}

func (row *csvRow) segmentID(field string, kind gmns.IDType) gmns.SegmentID {
	value, err := gmns.ParseSegmentID(row.required(field), kind)
	row.fail(field, err)
	return value
}

func (row *csvRow) segmentLaneID(field string, kind gmns.IDType) gmns.SegmentLaneID {
	value, err := gmns.ParseSegmentLaneID(row.required(field), kind)
	row.fail(field, err)
	return value
}
