package csvimport

import "github.com/LdDl/go-gmns/gmns"

func readLane(row *csvRow, kind gmns.IDType) *gmns.Lane {
	record := gmns.NewLane(row.laneID("lane_id", kind), row.linkID("link_id", kind), row.integer("lane_num"),
		gmns.WithLaneAllowedUses(row.text("allowed_uses")), gmns.WithLaneRightBarrier(row.text("r_barrier")), gmns.WithLaneLeftBarrier(row.text("l_barrier")))
	if row.text("width") != "" {
		gmns.WithLaneWidthMeters(row.number("width"))(record)
	}
	return record
}

func readSegmentLane(row *csvRow, kind gmns.IDType) *gmns.SegmentLane {
	record := gmns.NewSegmentLane(row.segmentLaneID("segment_lane_id", kind), row.segmentID("segment_id", kind), row.integer("lane_num"),
		gmns.WithSegmentLaneAllowedUses(row.text("allowed_uses")), gmns.WithSegmentLaneRightBarrier(row.text("r_barrier")), gmns.WithSegmentLaneLeftBarrier(row.text("l_barrier")))
	if row.text("width") != "" {
		gmns.WithSegmentLaneWidthMeters(row.number("width"))(record)
	}
	if row.text("parent_lane_id") != "" {
		gmns.WithSegmentLaneParentLane(row.laneID("parent_lane_id", kind))(record)
	}
	return record
}

func readSegment(row *csvRow, kind gmns.IDType) *gmns.Segment {
	record := gmns.NewSegment(row.segmentID("segment_id", kind), row.linkID("link_id", kind), row.nodeID("ref_node_id", kind), row.number("start_lr"), row.number("end_lr"))
	if row.text("grade") != "" {
		gmns.WithSegmentGrade(row.number("grade"))(record)
	}
	if row.text("capacity") != "" {
		gmns.WithSegmentCapacity(row.number("capacity"))(record)
	}
	if row.text("free_speed") != "" {
		gmns.WithSegmentFreeSpeed(row.number("free_speed"))(record)
	}
	if row.text("lanes") != "" {
		gmns.WithSegmentLanesNum(row.integer("lanes"))(record)
	}
	if row.text("l_lanes_added") != "" {
		gmns.WithSegmentLeftLanesAdded(row.integer("l_lanes_added"))(record)
	}
	if row.text("r_lanes_added") != "" {
		gmns.WithSegmentRightLanesAdded(row.integer("r_lanes_added"))(record)
	}
	if row.text("bike_facility") != "" {
		gmns.WithSegmentBikeFacility(row.text("bike_facility"))(record)
	}
	if row.text("ped_facility") != "" {
		gmns.WithSegmentPedFacility(row.text("ped_facility"))(record)
	}
	if row.text("parking") != "" {
		gmns.WithSegmentParking(row.text("parking"))(record)
	}
	if row.text("allowed_uses") != "" {
		gmns.WithSegmentAllowedUses(row.text("allowed_uses"))(record)
	}
	if row.text("toll") != "" {
		gmns.WithSegmentToll(row.number("toll"))(record)
	}
	if row.text("jurisdiction") != "" {
		gmns.WithSegmentJurisdiction(row.text("jurisdiction"))(record)
	}
	if row.text("row_width") != "" {
		gmns.WithSegmentRightOfWayWidthMeters(row.number("row_width"))(record)
	}
	return record
}
