package meso

import (
	"github.com/LdDl/osm2gmns/gmns"
)

type Net struct {
	Nodes map[gmns.NodeID]*Node
	Links map[gmns.LinkID]*Link
}
