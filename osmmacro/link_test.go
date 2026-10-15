package osmmacro

import (
	"testing"

	"github.com/LdDl/go-gmns/gmns"

	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/stretchr/testify/assert"
)

type linkValuesCase struct {
	name         string
	capacity     int
	freeSpeed    float64
	tagMaxSpeed  float64
	wantCapacity int
	wantSpeed    float64
}

func TestNewLinkPreservesSuppliedValues(t *testing.T) {
	cases := []linkValuesCase{
		{
			name:         "supplied values override defaults and speed tag",
			capacity:     1234,
			freeSpeed:    37.25,
			tagMaxSpeed:  60,
			wantCapacity: 1234,
			wantSpeed:    37.25,
		},
		{
			name:         "explicit zeros",
			capacity:     0,
			freeSpeed:    0,
			tagMaxSpeed:  60,
			wantCapacity: 0,
			wantSpeed:    0,
		},
		{
			name:         "capacity default with supplied speed",
			capacity:     -1,
			freeSpeed:    42.5,
			tagMaxSpeed:  -1,
			wantCapacity: types.NewCapacityDefault(types.LINK_RESIDENTIAL),
			wantSpeed:    42.5,
		},
		{
			name:         "supplied capacity with speed from tag",
			capacity:     1234,
			freeSpeed:    -1,
			tagMaxSpeed:  55,
			wantCapacity: 1234,
			wantSpeed:    55,
		},
		{
			name:         "road class defaults",
			capacity:     -1,
			freeSpeed:    -1,
			tagMaxSpeed:  -1,
			wantCapacity: types.NewCapacityDefault(types.LINK_RESIDENTIAL),
			wantSpeed:    types.NewSpeedDefault(types.LINK_RESIDENTIAL),
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			way := &wrappers.WayOSM{
				ID:        10,
				Capacity:  testCase.capacity,
				FreeSpeed: testCase.freeSpeed,
				LinkType:  types.LINK_RESIDENTIAL,
				Tags: wrappers.WayTags{
					Lanes:         4,
					LanesForward:  2,
					LanesBackward: 2,
					MaxSpeed:      testCase.tagMaxSpeed,
				},
			}
			nodes := []*wrappers.NodeOSM{
				{
					InnerNode: osm.Node{
						ID:  1,
						Lon: 30,
						Lat: 60,
					},
				},
				{
					InnerNode: osm.Node{
						ID:  2,
						Lon: 30.002,
						Lat: 60.001,
					},
				},
			}
			before := *way
			forward, err := NewLinkFrom(gmns.LinkIDFromInt64(1), gmns.NodeIDFromInt64(1), gmns.NodeIDFromInt64(2), 1, 2, macro.DIRECTION_FORWARD, way, nodes)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			backward, err := NewLinkFrom(gmns.LinkIDFromInt64(2), gmns.NodeIDFromInt64(2), gmns.NodeIDFromInt64(1), 2, 1, macro.DIRECTION_BACKWARD, way, nodes)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			for _, link := range []*macro.Link{forward, backward} {
				capacity, hasCapacity := link.Capacity()
				assert.True(t, hasCapacity)
				assert.Equal(t, float64(testCase.wantCapacity), capacity, "capacity must not be scaled by lane count")
				speed, hasSpeed := link.FreeSpeed()
				assert.True(t, hasSpeed)
				assert.Equal(t, testCase.wantSpeed, speed)
				lanes, hasLanes := link.LanesNum()
				assert.True(t, hasLanes)
				assert.Equal(t, 2, lanes)
			}
			assert.Equal(t, before, *way, "link preparation must not replace supplied way values")
		})
	}
}
