package expmacro

import (
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/go-gmns/utils/geomath"
	"github.com/LdDl/osm2gmns/wrappers"
)

func NewNodeFrom(id gmns.NodeID, node *wrappers.NodeOSM) *macro.Node {
	return macro.NewNodeFrom(
		id,
		macro.WithNodeName(node.Name),
		macro.WithOSMHighwayTag(node.OsmData.Highway),
		macro.WithOSMNodeID(node.ID),
		macro.WithNodeControlType(node.ControlType),
		macro.WithPointGeom(node.InnerNode.Point()),
		macro.WithPointGeomEuclidean(geomath.PointToEuclidean(node.InnerNode.Point())),
	)
}
