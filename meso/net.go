package meso

import (
	"fmt"

	"github.com/LdDl/go-gmns/gmns"
)

var (
	ErrLinkNotFound = fmt.Errorf("Link not found")
	ErrNodeNotFound = fmt.Errorf("Node not found")
)

type Net struct {
	Nodes map[gmns.NodeID]*Node
	Links map[gmns.LinkID]*Link
}
