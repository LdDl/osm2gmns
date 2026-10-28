package osmmacro

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestNodeElevationErrorsAreReportedWithoutEvidence(t *testing.T) {
	var warnings bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&warnings)
	t.Cleanup(func() {
		log.Logger = previous
	})
	nodes := geometryTestNodes()
	nodes[1].InnerNode.Tags = osm.Tags{{Key: "ele", Value: "unknown"}}
	nodes[2].InnerNode.Tags = osm.Tags{{Key: "ele", Value: "12 ft"}}
	nodes[3].InnerNode.Tags = osm.Tags{{Key: "ele", Value: "-0"}}
	way := &wrappers.WayOSM{
		ID:    10,
		Nodes: []osm.NodeID{1, 2, 3, 4, 1, 5, 3},
	}
	ways := []*wrappers.WayOSM{way}
	net, err := NewNetFromOSM(ways, nodes)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 2, strings.Count(warnings.String(), "unsupported_elevation"))
	assert.Contains(t, warnings.String(), `"osm_node_id":1`)
	assert.Contains(t, warnings.String(), `"osm_node_id":2`)
	for _, node := range net.Nodes {
		value, present := node.AltitudeMeters()
		if node.OSMNode() == 1 {
			assert.False(t, present)
		} else {
			assert.True(t, present)
			assert.True(t, math.Signbit(value))
		}
	}
	warnings.Reset()
	withEvidence, evidence, err := NewNetFromOSMWithEvidence(ways, nodes)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, net, withEvidence)
	assert.Empty(t, warnings.String(), "the evidence caller controls warning output")
	diagnostics := evidence.Diagnostics()
	if !assert.Len(t, diagnostics, 2) {
		return
	}
	assert.Equal(t, "1", diagnostics[0].ID)
	assert.Equal(t, "2", diagnostics[1].ID)
	for _, diagnostic := range diagnostics {
		assert.Equal(t, "unsupported_elevation", diagnostic.Code)
		assert.NotEmpty(t, diagnostic.Message)
	}
	// A later conversion must read its current inputs, not an old per-node cache.
	nodes[1].InnerNode.Tags[0].Value = "7"
	updated, err := NewNetFromOSM(ways, nodes)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 1, strings.Count(warnings.String(), "unsupported_elevation"))
	for _, node := range updated.Nodes {
		if node.OSMNode() == 1 {
			value, present := node.AltitudeMeters()
			assert.True(t, present)
			assert.Equal(t, 7.0, value)
		}
	}
}

func TestNewNodeFromReportsInvalidElevation(t *testing.T) {
	var warnings bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&warnings)
	t.Cleanup(func() {
		log.Logger = previous
	})
	nodes := geometryTestNodes()
	nodes[1].InnerNode.Tags = osm.Tags{{Key: "ele", Value: "unknown"}}
	node := NewNodeFrom(gmns.NodeIDFromInt64(1), nodes[1])
	_, present := node.AltitudeMeters()
	assert.False(t, present)
	assert.Equal(t, 1, strings.Count(warnings.String(), "unsupported_elevation"))
	assert.Contains(t, warnings.String(), `"osm_node_id":1`)
}
