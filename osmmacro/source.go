package osmmacro

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

// SourceDiagnostic describes evidence that could not establish standard vertical data.
type SourceDiagnostic struct {
	Kind    string
	ID      string
	Code    string
	Message string
}

type sourceWay struct {
	id    osm.WayID
	nodes []osm.NodeID
	tags  osm.Tags
}

type sourceLink struct {
	id        gmns.LinkID
	way       osm.WayID
	start     int
	end       int
	direction int
	from      gmns.NodeID
	to        gmns.NodeID
	shape     orb.LineString
}

type sourceNode struct {
	id    osm.NodeID
	node  gmns.NodeID
	point orb.Point
	tags  osm.Tags
}

// SourceEvidence is an immutable acquisition snapshot, independent of GMNS records.
// It is not a source of network edits or a replacement for standard tables.
type SourceEvidence struct {
	ways        map[osm.WayID]sourceWay
	links       map[gmns.LinkID]sourceLink
	nodes       []sourceNode
	diagnostics []SourceDiagnostic
}

func newSourceEvidence() *SourceEvidence {
	return &SourceEvidence{
		ways:  make(map[osm.WayID]sourceWay),
		links: make(map[gmns.LinkID]sourceLink),
	}
}

func (evidence *SourceEvidence) captureWay(way *wrappers.WayOSM) {
	_, exists := evidence.ways[way.ID]
	if exists {
		return
	}
	tags := way.Tags.SourceTags()
	evidence.ways[way.ID] = sourceWay{
		id:    way.ID,
		nodes: append([]osm.NodeID(nil), way.Nodes...),
		tags:  tags,
	}
	for _, message := range wayDiagnostics(tags) {
		evidence.diagnostics = append(evidence.diagnostics, SourceDiagnostic{
			Kind:    "way",
			ID:      strconv.FormatInt(int64(way.ID), 10),
			Code:    message.Code,
			Message: message.Message,
		})
	}
}

func (evidence *SourceEvidence) captureLink(link *macro.Link, way *wrappers.WayOSM, start, end, direction int) {
	evidence.links[link.ID] = sourceLink{
		id:        link.ID,
		way:       way.ID,
		start:     start,
		end:       end,
		direction: direction,
		from:      link.SourceNode(),
		to:        link.TargetNode(),
		shape:     link.Geom().Clone(),
	}
}

func (evidence *SourceEvidence) captureNodes(net *macro.Net, nodes map[osm.NodeID]*wrappers.NodeOSM) {
	identities := make(map[osm.NodeID]gmns.NodeID)
	for id, node := range net.Nodes {
		identities[node.OSMNode()] = id
	}
	used := make(map[osm.NodeID]bool)
	for _, way := range evidence.ways {
		for _, id := range way.nodes {
			used[id] = true
		}
	}
	ids := make([]osm.NodeID, 0, len(used))
	for id := range used {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(first, second int) bool {
		return ids[first] < ids[second]
	})
	for _, id := range ids {
		node := nodes[id]
		tags := wrappers.SourceTags(node.InnerNode.Tags)
		if len(tags) == 0 {
			continue
		}
		evidence.nodes = append(evidence.nodes, sourceNode{
			id:    id,
			node:  identities[id],
			point: node.InnerNode.Point(),
			tags:  tags,
		})
		_, present, err := wrappers.NodeElevation(tags)
		if err != nil {
			evidence.diagnostics = append(evidence.diagnostics, SourceDiagnostic{
				Kind:    "node",
				ID:      strconv.FormatInt(int64(id), 10),
				Code:    "unsupported_elevation",
				Message: err.Error(),
			})
		} else if present && identities[id].IsZero() {
			evidence.diagnostics = append(evidence.diagnostics, SourceDiagnostic{
				Kind:    "node",
				ID:      strconv.FormatInt(int64(id), 10),
				Code:    "shape_point_elevation",
				Message: "elevation retained as evidence; a 2D shape point is not a GMNS node",
			})
		}
		for _, tag := range tags {
			if len(tag.Key) > 4 && tag.Key[:4] == "ele:" && tag.Key != "ele:datum" {
				evidence.diagnostics = append(evidence.diagnostics, SourceDiagnostic{
					Kind:    "node",
					ID:      strconv.FormatInt(int64(id), 10),
					Code:    "alternate_elevation",
					Message: "alternate elevation retained without unit or datum conversion",
				})
				break
			}
		}
	}
}

// Diagnostics returns a copy; unknown heights never become zeroes or default profiles.
func (evidence *SourceEvidence) Diagnostics() []SourceDiagnostic {
	return append([]SourceDiagnostic(nil), evidence.diagnostics...)
}

// Attach serializes captured evidence after checking exact network identity and geometry.
// It never infers source spans or changes standard records. Omit this call for older strict readers.
func (evidence *SourceEvidence) Attach(net *macro.Net, dataset *csvexport.Dataset) error {
	if evidence == nil || net == nil || dataset == nil || dataset.Level != "macro" || dataset.Provenance == nil {
		return fmt.Errorf("source evidence requires a macro network and dataset")
	}
	if len(evidence.links) != len(net.Links) {
		return fmt.Errorf("source evidence no longer matches the network links")
	}
	ids := make([]gmns.LinkID, 0, len(evidence.links))
	for id, record := range evidence.links {
		link := net.Links[id]
		if link == nil || link.SourceNode() != record.from || link.TargetNode() != record.to || !link.Geom().Equal(record.shape) {
			return fmt.Errorf("link %v: source evidence no longer matches the network", id)
		}
		from := net.Nodes[record.from]
		to := net.Nodes[record.to]
		if from == nil || to == nil || from.Geom() != record.shape[0] || to.Geom() != record.shape[len(record.shape)-1] {
			return fmt.Errorf("link %v: source evidence no longer matches the endpoints", id)
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(first, second int) bool {
		return ids[first].Less(ids[second])
	})
	links := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		record := evidence.links[id]
		links = append(links, map[string]any{
			"link_id":     id.String(),
			"osm_way_id":  strconv.FormatInt(int64(record.way), 10),
			"start_index": record.start,
			"end_index":   record.end,
			"direction":   record.direction,
		})
	}
	wayIDs := make([]osm.WayID, 0, len(evidence.ways))
	for id := range evidence.ways {
		wayIDs = append(wayIDs, id)
	}
	sort.Slice(wayIDs, func(first, second int) bool {
		return wayIDs[first] < wayIDs[second]
	})
	ways := make([]map[string]any, 0, len(wayIDs))
	for _, id := range wayIDs {
		record := evidence.ways[id]
		refs := make([]string, len(record.nodes))
		for index, node := range record.nodes {
			refs[index] = strconv.FormatInt(int64(node), 10)
		}
		ways = append(ways, map[string]any{
			"osm_way_id": strconv.FormatInt(int64(id), 10),
			"node_refs":  refs,
			"tags":       sourceTagRecords(record.tags),
		})
	}
	nodes := make([]map[string]any, 0, len(evidence.nodes))
	for _, record := range evidence.nodes {
		item := map[string]any{
			"osm_node_id": strconv.FormatInt(int64(record.id), 10),
			"x_coord":     record.point[0],
			"y_coord":     record.point[1],
			"tags":        sourceTagRecords(record.tags),
		}
		if !record.node.IsZero() {
			item["node_id"] = record.node.String()
		}
		nodes = append(nodes, item)
	}
	diagnostics := make([]map[string]any, 0, len(evidence.diagnostics))
	for _, item := range evidence.diagnostics {
		diagnostics = append(diagnostics, map[string]any{
			"kind":    item.Kind,
			"id":      item.ID,
			"code":    item.Code,
			"message": item.Message,
		})
	}
	dataset.Provenance["osm_source"] = map[string]any{
		"format":        "osm2gmns-osm-source",
		"version":       1,
		"ele_units":     "meter",
		"ele_reference": "EGM96 under the OSM ele convention; no datum conversion or measurement verification",
		"ways":          ways,
		"links":         links,
		"nodes":         nodes,
		"diagnostics":   diagnostics,
	}
	return nil
}

func sourceTagRecords(tags osm.Tags) []map[string]string {
	result := make([]map[string]string, 0, len(tags))
	for _, tag := range tags {
		result = append(result, map[string]string{"key": tag.Key, "value": tag.Value})
	}
	return result
}
