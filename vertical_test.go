package osm2gmns_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/csvimport"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/preparation"
	"github.com/stretchr/testify/assert"
)

const elevationOSM = `<osm version="0.6">
<node id="1" lat="55" lon="37"><tag k="ele" v="-0"/></node>
<node id="2" lat="55" lon="37.001"><tag k="ele" v="21.25"/></node>
<node id="3" lat="55" lon="37.002"><tag k="ele" v="-12.5"/></node>
<node id="4" lat="55.001" lon="37.002"><tag k="ele" v="unknown"/></node>
<node id="5" lat="55.002" lon="37.002"/>
<way id="10"><nd ref="1"/><nd ref="2"/><nd ref="3"/><tag k="highway" v="residential"/><tag k="bridge" v="yes"/><tag k="layer" v="1"/><tag k="lanes" v="2"/></way>
<way id="11"><nd ref="3"/><nd ref="4"/><tag k="highway" v="residential"/><tag k="tunnel" v="building_passage"/><tag k="incline" v="up"/></way>
<way id="12"><nd ref="4"/><nd ref="5"/><tag k="highway" v="residential"/><tag k="layer" v="-2"/></way>
</osm>`

func TestOSMElevationAndStandardRoundTrip(t *testing.T) {
	input := filepath.Join(t.TempDir(), "elevations.osm")
	assert.NoError(t, os.WriteFile(input, []byte(elevationOSM), 0600))
	parser := osm2gmns.NewParser(input, osm2gmns.WithAllowedAgentTypes([]types.AgentType{types.AGENT_AUTO}))
	data, err := parser.ReadOSM()
	if !assert.NoError(t, err) {
		return
	}
	net, evidence, err := osm2gmns.GenerateMacroscopicWithEvidence(data, false)
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, preparation.Macro(net, preparation.DefaultUseOptions()))
	for _, mode := range []osmmacro.GeometryMode{osmmacro.GEOMETRY_INLINE, osmmacro.GEOMETRY_SHARED} {
		dataset, buildErr := osmmacro.BuildDataset(net, mode)
		assert.NoError(t, buildErr)
		assert.NoError(t, evidence.Attach(net, &dataset))
		directory := t.TempDir()
		assert.NoError(t, csvexport.WriteDatasets(directory, []csvexport.Dataset{dataset}))
		// A producer-independent reader must not need even valid application metadata.
		assert.NoError(t, os.WriteFile(filepath.Join(directory, "provenance_macro.json"), []byte("invalid optional JSON"), 0600))
		restored, readErr := csvimport.ReadMacro(os.DirFS(directory), "datapackage_macro.json")
		if !assert.NoError(t, readErr) {
			return
		}
		for id, source := range net.Nodes {
			value, present := restored.Nodes[id].AltitudeMeters()
			expected, supplied := source.AltitudeMeters()
			assert.Equal(t, supplied, present)
			assert.Equal(t, math.Float64bits(expected), math.Float64bits(value))
			switch source.OSMNode() {
			case 1:
				assert.True(t, present)
				assert.True(t, math.Signbit(value))
			case 3:
				assert.True(t, present)
				assert.Equal(t, -12.5, value)
			default:
				assert.False(t, present)
			}
		}
		for _, link := range restored.Links {
			_, supplied := link.Grade()
			assert.False(t, supplied)
		}
		// Altitude survives cloning independently of the OSM source snapshot.
		clone := restored.Clone()
		assert.Equal(t, restored.Nodes[gmns.NodeIDFromInt64(0)].Geom(), clone.Nodes[gmns.NodeIDFromInt64(0)].Geom())
	}
}
