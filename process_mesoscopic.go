package osm2gmns

import (
	"math"
	"time"

	"github.com/LdDl/osm2gmns/geomath"
	"github.com/LdDl/osm2gmns/gmns"
	"github.com/LdDl/osm2gmns/macro"
	"github.com/LdDl/osm2gmns/meso"
	"github.com/LdDl/osm2gmns/movement"
	"github.com/LdDl/osm2gmns/types"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	SHORTCUT_LENGTH  = 0.1
	MIN_CUT_LENGTH   = 2.0
	TOTAL_CUT_LENGTH = 2 * SHORTCUT_LENGTH * MIN_CUT_LENGTH
)

var (
	CUT_LENGTHS = [100]float64{2.0, 8.0, 12.0, 14.0, 16.0, 18.0, 20, 22, 24, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25}
)

type macroLinkProcessing struct {
	needsOffset         bool
	id                  gmns.LinkID
	offsetGeomEuclidean orb.LineString
	offsetGeom          orb.LineString
	lanesInfo           macro.LanesInfo
	lengthMetersOffset  float64

	downstreamShortCut bool
	upstreamShortCut   bool

	downstreamIsTarget bool
	upstreamIsTarget   bool

	upstreamCutLen   float64
	downstreamCutLen float64
}

func GenerateMesoscopic(macroNet *macro.Net, movements movement.MovementsStorage) (*meso.Net, error) {
	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Msg("Preparing mesoscopic network")
	}
	var mesoNet meso.Net
	st := time.Now()
	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Msg("Preparing geometries offsets")
	}
	macroLinks := macroLinksToSlice(macroNet.Links)
	needToObserve := make(map[gmns.LinkID]*macroLinkProcessing, len(macroLinks))
	for i := range macroLinks {
		macroLink := macroLinks[i]
		macroLinkID := macroLink.ID
		if _, ok := needToObserve[macroLinkID]; ok {
			continue
		}
		reversedGeom := macroLink.GeomEuclidean().Clone() // Clone to prevent changing macroscopic link Euclidean geometry
		reversedGeom.Reverse()
		reversedLinkExists := false
		// Scan other links
		for j := range macroLinks[i+1:] {
			macroLinkCompare := macroLinks[i+1:][j]
			macroLinkCompareID := macroLinkCompare.ID
			if orb.Equal(reversedGeom, macroLinkCompare.GeomEuclidean()) {
				reversedLinkExists = true
				needToObserve[macroLinkID] = &macroLinkProcessing{id: macroLinkID, lanesInfo: macroLink.LanesInfo(), needsOffset: true}
				needToObserve[macroLinkCompareID] = &macroLinkProcessing{id: macroLinkCompareID, lanesInfo: macroLinkCompare.LanesInfo(), needsOffset: true}
				break
			}
		}
		if !reversedLinkExists {
			needToObserve[macroLinkID] = &macroLinkProcessing{id: macroLinkID, lanesInfo: macroLink.LanesInfo()}
		}
	}

	for macroLinkID := range needToObserve {
		macroLinkProcess := needToObserve[macroLinkID]
		macroLink, ok := macroNet.Links[macroLinkID]
		if !ok {
			return nil, errors.Wrapf(macro.ErrLinkNotFound, "Offset Link ID: %d", macroLinkID)
		}
		if !macroLinkProcess.needsOffset {
			macroLinkProcess.offsetGeomEuclidean = macroLink.GeomEuclidean().Clone()
			macroLinkProcess.offsetGeom = macroLink.Geom().Clone()
			continue
		}
		offsetDistance := 2 * (float64(macroLink.MaxLanes())/2 + 0.5) * macro.LANE_WIDTH
		macroLinkProcess.offsetGeomEuclidean = geomath.OffsetCurve(macroLink.GeomEuclidean(), -offsetDistance)
		macroLinkProcess.offsetGeom = geomath.LineToSpherical(macroLinkProcess.offsetGeomEuclidean)
	}
	// Update breakpoints since geometry has changed
	for macroLinkID := range needToObserve {
		macroLinkProcess := needToObserve[macroLinkID]
		// Re-calcuate length for offset geometry and round to 2 decimal places
		macroLinkProcess.lengthMetersOffset = math.Round(geo.LengthHaversine(macroLinkProcess.offsetGeom)*100.0) / 100.0
		macroLink, ok := macroNet.Links[macroLinkID]
		if !ok {
			return nil, errors.Wrapf(macro.ErrLinkNotFound, "Offset Link ID: %d", macroLinkID)
		}
		for i, item := range macroLinkProcess.lanesInfo.LanesChangePoints {
			macroLinkProcess.lanesInfo.LanesChangePoints[i] = (item / macroLink.LengthMeters()) * macroLinkProcess.lengthMetersOffset
		}
	}
	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Msg("Aggregate movements for nodes")
	}
	macroNodesMovements := make(map[gmns.NodeID][]*movement.Movement, len(macroNet.Nodes))
	for i := range movements {
		mvmt := movements[i]
		if _, ok := macroNet.Nodes[mvmt.MacroNodeID]; !ok {
			return nil, errors.Wrapf(macro.ErrNodeNotFound, "Agg movements; Node ID: %d", mvmt.MacroNodeID)
		}
		if _, ok := macroNodesMovements[mvmt.MacroNodeID]; !ok {
			macroNodesMovements[mvmt.MacroNodeID] = make([]*movement.Movement, 0, 1)
		}
		macroNodesMovements[mvmt.MacroNodeID] = append(macroNodesMovements[mvmt.MacroNodeID], mvmt)
	}

	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Msg("Process movements (check necessity)")
	}

	macroNodesNeedMovement := make(map[gmns.NodeID]bool)
	// Assume that every node need movement (i.e. all nodes are intersections by default). We will filter this set later
	for i := range macroNet.Nodes {
		macroNodesNeedMovement[macroNet.Nodes[i].ID] = true
	}

	for i := range macroNet.Nodes {
		macroNode := macroNet.Nodes[i]
		if macroNode.ControlType() == types.CONTROL_TYPE_IS_SIGNAL {
			continue
		}
		incomingMacroLinks := macroNode.IncomingLinks()
		outcomingMacroLinks := macroNode.OutcomingLinks()
		if len(incomingMacroLinks) == 1 && len(outcomingMacroLinks) >= 1 {
			// Only one incoming link
			incomingMacroLinkID := incomingMacroLinks[0]
			incomingMacroLink, ok := macroNet.Links[incomingMacroLinkID]
			if !ok {
				return nil, errors.Wrapf(macro.ErrLinkNotFound, "Case 1: Incoming link ID: %d for macroscopic node %d", incomingMacroLinkID, macroNode.ID)
			}
			badAngle := true
			for j := range outcomingMacroLinks {
				outcomingMacroLinkID := outcomingMacroLinks[j]
				outcomingMacroLink, ok := macroNet.Links[outcomingMacroLinkID]
				if !ok {
					return nil, errors.Wrapf(macro.ErrLinkNotFound, "Case 1: Outcoming link ID: %d for macroscopic node %d", outcomingMacroLinkID, macroNode.ID)
				}
				angle := geomath.AngleBetweenLines(incomingMacroLink.GeomEuclidean(), outcomingMacroLink.GeomEuclidean())
				if angle > 0.75*math.Pi || angle < -0.75*math.Pi {
					badAngle = false
					break
				}
			}
			if !badAngle {
				continue
			}
			hasMultipleConnections := false
			outcomingMacroLinksObserved := make(map[gmns.LinkID]struct{})
			macroNodeMvmts, ok := macroNodesMovements[macroNode.ID]
			if ok {
				for j := range macroNodeMvmts {
					mvmt := macroNodeMvmts[j]
					if _, ok := outcomingMacroLinksObserved[mvmt.OutcomeMacroLinkID]; ok {
						hasMultipleConnections = true
						break
					}
					outcomingMacroLinksObserved[mvmt.OutcomeMacroLinkID] = struct{}{}
				}
			}
			if hasMultipleConnections {
				continue
			}
			macroNodesNeedMovement[macroNode.ID] = false
			macroLinkProcess := needToObserve[incomingMacroLinkID]
			if macroLinkProcess == nil {
				panic("Should find incoming macroscopic link in observable data")
			}
			macroLinkProcess.downstreamShortCut = true
			macroLinkProcess.downstreamIsTarget = true
			for j := range outcomingMacroLinks {
				outcomingMacroLinkID := outcomingMacroLinks[j]
				outcomingMacroLink, ok := needToObserve[outcomingMacroLinkID]
				if !ok {
					return nil, errors.Wrapf(macro.ErrLinkNotFound, "Nested outcoming link ID: %d for macroscopic node %d", outcomingMacroLinkID, macroNode.ID)
				}
				outcomingMacroLink.upstreamShortCut = true
			}
		} else if len(incomingMacroLinks) >= 1 && len(outcomingMacroLinks) == 1 {
			// Only one outcoming link
			outcomingMacroLinkID := outcomingMacroLinks[0]
			outcomingMacroLink, ok := macroNet.Links[outcomingMacroLinkID]
			if !ok {
				return nil, errors.Wrapf(macro.ErrLinkNotFound, "Case 2: Outcoming link ID: %d for macroscopic node %d", outcomingMacroLinkID, macroNode.ID)
			}
			badAngle := true
			for j := range incomingMacroLinks {
				incomingMacroLinkID := incomingMacroLinks[j]
				incomingMacroLink, ok := macroNet.Links[incomingMacroLinkID]
				if !ok {
					return nil, errors.Wrapf(macro.ErrLinkNotFound, "Case 2: Incoming link ID: %d for macroscopic node %d", incomingMacroLinkID, macroNode.ID)
				}
				angle := geomath.AngleBetweenLines(incomingMacroLink.GeomEuclidean(), outcomingMacroLink.GeomEuclidean())
				if angle > 0.75*math.Pi || angle < -0.75*math.Pi {
					badAngle = false
					break
				}
			}
			if !badAngle {
				continue
			}
			hasMultipleConnections := false
			outcomingMacroLinksObserved := make(map[gmns.LinkID]struct{})
			macroNodeMvmts, ok := macroNodesMovements[macroNode.ID]
			if ok {
				for j := range macroNodeMvmts {
					mvmt := macroNodeMvmts[j]
					if _, ok := outcomingMacroLinksObserved[mvmt.IncomeMacroLinkID]; ok {
						hasMultipleConnections = true
						break
					}
					outcomingMacroLinksObserved[mvmt.IncomeMacroLinkID] = struct{}{}
				}
			}
			if hasMultipleConnections {
				continue
			}
			macroNodesNeedMovement[macroNode.ID] = false
			macroLinkProcess := needToObserve[outcomingMacroLinkID]
			if macroLinkProcess == nil {
				panic("Should find outcoming macroscopic link in observable data")
			}
			macroLinkProcess.upstreamShortCut = true
			macroLinkProcess.upstreamIsTarget = true
			for j := range incomingMacroLinks {
				incomingMacroLinkID := incomingMacroLinks[j]
				incomingMacroLink, ok := needToObserve[incomingMacroLinkID]
				if !ok {
					return nil, errors.Wrapf(macro.ErrLinkNotFound, "Nested incoming link ID: %d for macroscopic node %d", incomingMacroLinkID, macroNode.ID)
				}
				incomingMacroLink.downstreamShortCut = true
			}
		}
	}

	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Msg("Process movements (calculate cuts' lengths and perform cuts)")
	}

	for macroLinkID := range needToObserve {
		macroLinkProcess := needToObserve[macroLinkID]
		macroLinkProcess.updateCutLength()
		panic("@todo")
		macroLinkProcess.performCut()
	}

	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Int("macro_nodes_num", len(mesoNet.Nodes)).Int("macro_links_num", len(mesoNet.Links)).Float64("elapsed", time.Since(st).Seconds()).Msg("Preparing mesoscopic network done!")
	}
	return nil, nil
}

func macroLinksToSlice(links map[gmns.LinkID]*macro.Link) []*macro.Link {
	ans := make([]*macro.Link, 0, len(links))
	for i := range links {
		ans = append(ans, links[i])
	}
	return ans
}

func (macroLinkProcess *macroLinkProcessing) updateCutLength() {
	laneChangePoints := macroLinkProcess.lanesInfo.LanesChangePoints
	// Dodge potential change of number of lanes on two ends of the macroscopic link
	// @todo: Bound check for LanesChangePoints
	upstreamMaxCut := math.Max(SHORTCUT_LENGTH, laneChangePoints[1]-laneChangePoints[0]-3)
	// Defife a variable downstreamMaxCut which is the maximum length of a cut that can be made downstream of the link,
	// calculated as the maximum of the shortcutLen and the difference between the last two elements in the link.lanesChangePoints minus 3.
	// @todo: Bound check for LanesChangePoints
	downstreamMaxCut := math.Max(SHORTCUT_LENGTH, laneChangePoints[len(laneChangePoints)-1]-laneChangePoints[len(laneChangePoints)-2]-3)
	if macroLinkProcess.upstreamShortCut && macroLinkProcess.downstreamShortCut {
		if macroLinkProcess.lengthMetersOffset > TOTAL_CUT_LENGTH {
			macroLinkProcess.upstreamCutLen = SHORTCUT_LENGTH
			macroLinkProcess.downstreamCutLen = SHORTCUT_LENGTH
		} else {
			macroLinkProcess.upstreamCutLen = (macroLinkProcess.lengthMetersOffset / TOTAL_CUT_LENGTH) * SHORTCUT_LENGTH
			macroLinkProcess.downstreamCutLen = macroLinkProcess.upstreamCutLen
		}
	} else if macroLinkProcess.upstreamShortCut {
		cutIdx := 0
		cutPlaceFound := false
		for i := macroLinkProcess.lanesInfo.LanesList[len(macroLinkProcess.lanesInfo.LanesList)-1]; i >= 0; i-- {
			if macroLinkProcess.lengthMetersOffset > math.Min(downstreamMaxCut, CUT_LENGTHS[i])+SHORTCUT_LENGTH+MIN_CUT_LENGTH {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			macroLinkProcess.upstreamCutLen = SHORTCUT_LENGTH
			macroLinkProcess.downstreamCutLen = math.Min(downstreamMaxCut, CUT_LENGTHS[cutIdx])
		} else {
			downStreamCut := math.Min(downstreamMaxCut, CUT_LENGTHS[0])
			totalLen := downStreamCut + SHORTCUT_LENGTH + MIN_CUT_LENGTH
			macroLinkProcess.upstreamCutLen = (macroLinkProcess.lengthMetersOffset / totalLen) * SHORTCUT_LENGTH
			macroLinkProcess.downstreamCutLen = (macroLinkProcess.lengthMetersOffset / totalLen) * downStreamCut
		}
	} else if macroLinkProcess.downstreamShortCut {
		cutIdx := 0
		cutPlaceFound := false
		for i := macroLinkProcess.lanesInfo.LanesList[len(macroLinkProcess.lanesInfo.LanesList)-1]; i >= 0; i-- {
			if macroLinkProcess.lengthMetersOffset > math.Min(upstreamMaxCut, CUT_LENGTHS[i])+SHORTCUT_LENGTH+MIN_CUT_LENGTH {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			macroLinkProcess.upstreamCutLen = math.Min(upstreamMaxCut, CUT_LENGTHS[cutIdx])
			macroLinkProcess.downstreamCutLen = SHORTCUT_LENGTH
		} else {
			upStreamCut := math.Min(upstreamMaxCut, CUT_LENGTHS[0])
			totalLen := upStreamCut + SHORTCUT_LENGTH + MIN_CUT_LENGTH
			macroLinkProcess.upstreamCutLen = (macroLinkProcess.lengthMetersOffset / totalLen) * CUT_LENGTHS[0]
			macroLinkProcess.downstreamCutLen = (macroLinkProcess.lengthMetersOffset / totalLen) * SHORTCUT_LENGTH
		}
	} else {
		cutIdx := 0
		cutPlaceFound := false
		for i := macroLinkProcess.lanesInfo.LanesList[len(macroLinkProcess.lanesInfo.LanesList)-1]; i >= 0; i-- {
			if macroLinkProcess.lengthMetersOffset > math.Min(upstreamMaxCut, CUT_LENGTHS[i])+math.Min(downstreamMaxCut, CUT_LENGTHS[i])+MIN_CUT_LENGTH {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			macroLinkProcess.upstreamCutLen = math.Min(upstreamMaxCut, CUT_LENGTHS[cutIdx])
			macroLinkProcess.downstreamCutLen = math.Min(downstreamMaxCut, CUT_LENGTHS[cutIdx])
		} else {
			upStreamCut := math.Min(upstreamMaxCut, CUT_LENGTHS[0])
			downStreamCut := math.Min(downstreamMaxCut, CUT_LENGTHS[0])
			totalLen := downStreamCut + upStreamCut + MIN_CUT_LENGTH
			macroLinkProcess.upstreamCutLen = (macroLinkProcess.lengthMetersOffset / totalLen) * upStreamCut
			macroLinkProcess.downstreamCutLen = (macroLinkProcess.lengthMetersOffset / totalLen) * downStreamCut
		}
	}
}

func (macroLinkProcess *macroLinkProcessing) performCut() {

}
