package osm2gmns

import (
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
)

type OSMScanner interface {
	Scan() bool
	Close() error
	Err() error
	Object() osm.Object
}

type OSMWaysNodes struct {
	nodes map[osm.NodeID]*Node
	ways  []*wrappers.WayOSM
}
