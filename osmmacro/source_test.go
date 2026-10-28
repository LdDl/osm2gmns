package osmmacro

import (
	"encoding/json"
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
	"github.com/stretchr/testify/assert"
)

func TestSourceEvidenceRetainsSplitReverseAndRepeatedSpans(t *testing.T) {
	nodes := geometryTestNodes()
	nodes[1].InnerNode.Tags = osm.Tags{{Key: "ele", Value: "0"}}
	nodes[2].InnerNode.Tags = osm.Tags{{Key: "ele", Value: "100"}}
	nodes[3].InnerNode.Tags = osm.Tags{{Key: "ele", Value: "-12.5"}}
	original := &osm.Way{
		ID: 10,
		Tags: osm.Tags{
			{Key: "highway", Value: "residential"},
			{Key: "bridge", Value: "yes"},
			{Key: "layer", Value: "1"},
			{Key: "incline", Value: "up"},
		},
	}
	for _, id := range []osm.NodeID{1, 2, 3, 4, 1, 5, 3} {
		original.Nodes = append(original.Nodes, osm.WayNode{ID: id})
	}
	way := wrappers.NewWayOSMFrom(original)
	ways := []*wrappers.WayOSM{way}
	net, evidence, err := NewNetFromOSMWithEvidence(ways, nodes)
	if !assert.NoError(t, err) {
		return
	}
	assert.Len(t, net.Links, 6)
	assert.Len(t, evidence.links, 6)
	for _, node := range net.Nodes {
		value, present := node.AltitudeMeters()
		assert.True(t, present)
		if node.OSMNode() == 1 {
			assert.Equal(t, 0.0, value)
		} else {
			assert.Equal(t, -12.5, value)
		}
	}
	for index := 0; index < 6; index++ {
		id := gmns.LinkIDFromInt64(int64(index))
		record := evidence.links[id]
		assert.Equal(t, (index/2)*2, record.start)
		assert.Equal(t, (index/2)*2+2, record.end)
		direction := 1
		if index%2 == 1 {
			direction = -1
		}
		assert.Equal(t, direction, record.direction)
	}
	first, err := BuildDataset(net, GEOMETRY_INLINE)
	assert.NoError(t, err)
	assert.NoError(t, evidence.Attach(net, &first))
	second, err := BuildDataset(net, GEOMETRY_SHARED)
	assert.NoError(t, err)
	assert.NoError(t, evidence.Attach(net, &second))
	assert.Equal(t, first.Provenance["osm_source"], second.Provenance["osm_source"])
	bytes, err := json.Marshal(first.Provenance["osm_source"])
	assert.NoError(t, err)
	assert.Contains(t, string(bytes), "shape_point_elevation")
	assert.Contains(t, string(bytes), "incline_not_mapped")
	assert.Contains(t, string(bytes), "100")
	// Changing the input wrappers after generation must not change the captured evidence.
	way.Nodes[0] = 999
	nodes[2].InnerNode.Tags[0].Value = "999"
	assert.NoError(t, evidence.Attach(net, &second))
	assert.Equal(t, first.Provenance["osm_source"], second.Provenance["osm_source"])
	link := net.Links[gmns.LinkIDFromInt64(0)]
	macro.WithLineGeom(orb.LineString{{37, 55}, {37.5, 55.5}})(link)
	assert.Error(t, evidence.Attach(net, &second))
}

func TestSourceEvidenceDiagnosticsKeepUnsupportedValues(t *testing.T) {
	tags := osm.Tags{
		{Key: "tunnel", Value: "building_passage"},
		{Key: "covered", Value: "yes"},
		{Key: "layer", Value: "-2;-1"},
		{Key: "layer", Value: "-2"},
		{Key: "oneway", Value: "-1"},
		{Key: "ele", Value: "20"},
	}
	diagnostics := wayDiagnostics(tags)
	codes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	assert.Contains(t, codes, "tunnel_classification")
	assert.Contains(t, codes, "covered_not_tunnel")
	assert.Contains(t, codes, "repeated_tag")
	assert.Contains(t, codes, "reverse_oneway_unsupported")
	assert.Contains(t, codes, "way_elevation_not_mapped")
}
