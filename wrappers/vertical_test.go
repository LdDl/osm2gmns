package wrappers

import (
	"math"
	"testing"

	"github.com/paulmach/osm"
	"github.com/stretchr/testify/assert"
)

type elevationCase struct {
	name    string
	tags    osm.Tags
	present bool
	value   float64
	invalid bool
}

func TestNodeElevation(t *testing.T) {
	cases := []elevationCase{
		{name: "absent"},
		{
			name:    "zero",
			tags:    osm.Tags{{Key: "ele", Value: "0"}},
			present: true,
		},
		{
			name:    "negative",
			tags:    osm.Tags{{Key: "ele", Value: "-12.75"}},
			present: true,
			value:   -12.75,
		},
		{
			name:    "meters",
			tags:    osm.Tags{{Key: "ele", Value: " 15.5 m "}},
			present: true,
			value:   15.5,
		},
		{
			name:    "empty",
			tags:    osm.Tags{{Key: "ele", Value: ""}},
			invalid: true,
		},
		{
			name:    "feet",
			tags:    osm.Tags{{Key: "ele", Value: "12 ft"}},
			invalid: true,
		},
		{
			name:    "infinity",
			tags:    osm.Tags{{Key: "ele", Value: "+Inf"}},
			invalid: true,
		},
		{
			name:    "nan",
			tags:    osm.Tags{{Key: "ele", Value: "NaN"}},
			invalid: true,
		},
		{
			name:    "overflow",
			tags:    osm.Tags{{Key: "ele", Value: "1e999"}},
			invalid: true,
		},
		{name: "local", tags: osm.Tags{{Key: "ele:local", Value: "2"}}},
		{
			name:    "datum",
			tags:    osm.Tags{{Key: "ele", Value: "2"}, {Key: "ele:datum", Value: "local"}},
			invalid: true,
		},
		{
			name:    "duplicate",
			tags:    osm.Tags{{Key: "ele", Value: "2"}, {Key: "ele", Value: "3"}},
			invalid: true,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			value, present, err := NodeElevation(testCase.tags)
			assert.Equal(t, testCase.present, present)
			assert.Equal(t, testCase.value, value)
			if testCase.invalid {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
	tags := osm.Tags{{Key: "ele", Value: "-0"}}
	value, present, err := NodeElevation(tags)
	assert.NoError(t, err)
	assert.True(t, present)
	assert.True(t, math.Signbit(value))
}

func TestSourceTagsPreserveExactEvidence(t *testing.T) {
	source := osm.Tags{
		{Key: "bridge", Value: "yes"},
		{Key: "layer", Value: ""},
		{Key: "layer", Value: "-2"},
		{Key: "tunnel", Value: "building_passage"},
		{Key: "ele:local", Value: "1"},
	}
	way := &osm.Way{Tags: source}
	tags := NewWayTagsFrom(way)
	assert.Equal(t, source, tags.SourceTags())
	copyTags := tags.SourceTags()
	copyTags[0].Value = "no"
	assert.Equal(t, source, tags.SourceTags())
}
