package osm2gmns

import (
	"fmt"
	"math"
	"sort"
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
	CUT_LENGTHS          = [100]float64{2.0, 8.0, 12.0, 14.0, 16.0, 18.0, 20, 22, 24, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25}
	ErrNotImplementedYet = fmt.Errorf("Not implemented yet")
)

type macroLinkProcessing struct {
	needsOffset         bool
	id                  gmns.LinkID
	offsetGeomEuclidean orb.LineString
	offsetGeom          orb.LineString
	lanesInfo           macro.LanesInfo
	lanesInfoCut        macro.LanesInfo
	lengthMetersOffset  float64

	/* For cuts */
	downstreamShortCut bool
	upstreamShortCut   bool

	downstreamIsTarget bool
	upstreamIsTarget   bool

	upstreamCutLen   float64
	downstreamCutLen float64

	offsetGeomEuclideanCut []orb.LineString
	offsetGeomCut          []orb.LineString

	/* For link generation */
	sourceMacroNodeID gmns.NodeID
	targetMacroNodeID gmns.NodeID
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
				needToObserve[macroLinkID] = &macroLinkProcessing{id: macroLinkID, lanesInfo: macroLink.LanesInfo(), needsOffset: true, sourceMacroNodeID: macroLink.SourceNode(), targetMacroNodeID: macroLink.TargetNode()}
				needToObserve[macroLinkCompareID] = &macroLinkProcessing{id: macroLinkCompareID, lanesInfo: macroLinkCompare.LanesInfo(), needsOffset: true, sourceMacroNodeID: macroLinkCompare.SourceNode(), targetMacroNodeID: macroLinkCompare.TargetNode()}
				break
			}
		}
		if !reversedLinkExists {
			needToObserve[macroLinkID] = &macroLinkProcessing{id: macroLinkID, lanesInfo: macroLink.LanesInfo(), sourceMacroNodeID: macroLink.SourceNode(), targetMacroNodeID: macroLink.TargetNode()}
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
		macroLinkProcess.performCut()
	}

	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Msg("Build mesoscopic links")
	}
	mesoNodes, mesoLinks, err := generateBaseNodesLinks(macroNet.Nodes, needToObserve)
	if err != nil {
		return nil, errors.Wrap(err, "Can't generate base mesoscopic nodes and links")
	}

	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Msg("Connect mesoscopic links")
	}

	err = connectMesoscopicLinks(mesoLinks, mesoNodes, macroNet.Nodes, macroNet.Links, macroNodesMovements, macroNodesNeedMovement)
	if err != nil {
		return nil, errors.Wrap(err, "Can't prepare connections between mesoscopic links")
	}

	panic("@todo")

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
	lanesInfo := macroLinkProcess.lanesInfo

	// Create copy for those since we will do mutations and want to keep original data
	lanesChangePoints := make([]float64, len(lanesInfo.LanesChangePoints))
	copy(lanesChangePoints, lanesInfo.LanesChangePoints)

	macroLinkProcess.lanesInfoCut.LanesList = make([]int, len(lanesInfo.LanesList))
	copy(macroLinkProcess.lanesInfoCut.LanesList, lanesInfo.LanesList)
	macroLinkProcess.lanesInfoCut.LanesChange = make([][2]int, len(lanesInfo.LanesChange))
	copy(macroLinkProcess.lanesInfoCut.LanesChange, lanesInfo.LanesChange)

	lanesChangePoints[0] = macroLinkProcess.upstreamCutLen
	lanesChangePoints[len(lanesChangePoints)-1] = macroLinkProcess.lengthMetersOffset - macroLinkProcess.downstreamCutLen
	// breakIdx := 1
	// for breakIdx = 1; breakIdx < len(lanesChangePoints); breakIdx++ {
	// 	if lanesChangePoints[breakIdx] > macroLinkProcess.upstreamCutLen {
	// 		break
	// 	}
	// }
	// lanesChangePoints = append(lanesChangePoints[breakIdx:])
	// lanesChangePoints = append([]float64{macroLinkProcess.upstreamCutLen}, lanesChangePoints...)
	// macroLinkProcess.lanesInfoCut.LanesList = macroLinkProcess.lanesInfoCut.LanesList[breakIdx-1:]
	// macroLinkProcess.lanesInfo.LanesChange = macroLinkProcess.lanesInfo.LanesChange[breakIdx-1:]

	// breakIdx = len(lanesChangePoints) - 2
	// for breakIdx := len(lanesChangePoints) - 2; breakIdx >= 0; breakIdx-- {
	// 	if macroLinkProcess.lengthMetersOffset-lanesChangePoints[breakIdx] > macroLinkProcess.downstreamCutLen {
	// 		break
	// 	}
	// }
	// lanesChangePoints = lanesChangePoints[:breakIdx+1]
	// lanesChangePoints = append(lanesChangePoints, macroLinkProcess.lengthMetersOffset-macroLinkProcess.downstreamCutLen)
	// macroLinkProcess.lanesInfoCut.LanesList = macroLinkProcess.lanesInfoCut.LanesList[:breakIdx+1]
	// macroLinkProcess.lanesInfo.LanesChange = macroLinkProcess.lanesInfo.LanesChange[:breakIdx+1]

	for i := range macroLinkProcess.lanesInfoCut.LanesList {
		start := lanesChangePoints[i]
		end := lanesChangePoints[i+1]
		geomCut := geomath.SubstringHaversine(macroLinkProcess.offsetGeom, start, end)
		geomEuclideanCut := geomath.LineToEuclidean(geomCut)
		macroLinkProcess.offsetGeomCut = append(macroLinkProcess.offsetGeomCut, geomCut)
		macroLinkProcess.offsetGeomEuclideanCut = append(macroLinkProcess.offsetGeomEuclideanCut, geomEuclideanCut)
	}
}

func generateBaseNodesLinks(macroNodes map[gmns.NodeID]*macro.Node, macroLinksProcessed map[gmns.LinkID]*macroLinkProcessing) (map[gmns.NodeID]*meso.Node, map[gmns.LinkID]*meso.Link, error) {
	lastMesoLinkID := gmns.LinkID(0)
	expandedMesoNodes := make(map[gmns.NodeID]int)
	collectedMesoNodes := make(map[gmns.NodeID]*meso.Node)
	collectedMesoLinks := make(map[gmns.LinkID]*meso.Link)
	for macroLinkID := range macroLinksProcessed {
		macroLinkProcess := macroLinksProcessed[macroLinkID]

		// Prepare source mesoscopic node
		var upstreamMesoNode *meso.Node
		sourceMacroNode, ok := macroNodes[macroLinkProcess.sourceMacroNodeID]
		if !ok {
			return nil, nil, errors.Wrapf(macro.ErrNodeNotFound, "Source node ID: %d", macroLinkProcess.sourceMacroNodeID)
		}
		if sourceMacroNode.IsCentroid() {
			// @todo: handle centroids
			return nil, nil, errors.Wrap(ErrNotImplementedYet, "Prepare upstream mesoscopic node from centroid")
		} else {
			expNodesNum, ok := expandedMesoNodes[macroLinkProcess.sourceMacroNodeID]
			if !ok {
				expandedMesoNodes[macroLinkProcess.sourceMacroNodeID] = 0
			}
			expandedMesoNodes[macroLinkProcess.sourceMacroNodeID] += 1
			upstreamMesoNode = meso.NewNodeFrom(
				macroLinkProcess.sourceMacroNodeID*100+gmns.NodeID(expNodesNum),
				meso.WithPointGeom(macroLinkProcess.offsetGeomCut[0][0]), // No explicit copy or clone method since Point is not slice, but array
				meso.WithPointEuclideanGeom(macroLinkProcess.offsetGeomEuclideanCut[0][0]),
				meso.WithPointMacroNode(macroLinkProcess.sourceMacroNodeID),
				meso.WithPointMacroLink(-1),
				meso.WithMacroZone(sourceMacroNode.Zone()),
				meso.WithActivityLinkType(sourceMacroNode.ActivityLinkType()),
				meso.WithBoundaryType(types.BOUNDARY_NONE),
			)
			collectedMesoNodes[upstreamMesoNode.ID] = upstreamMesoNode
		}

		// Prepare mesoscopic link and target mesoscopic node
		var downstreamMesoNode *meso.Node
		targetMacroNode, ok := macroNodes[macroLinkProcess.targetMacroNodeID]
		if !ok {
			return nil, nil, errors.Wrapf(macro.ErrNodeNotFound, "Target node ID: %d", macroLinkProcess.targetMacroNodeID)
		}
		segmentsToCut := len(macroLinkProcess.lanesInfoCut.LanesList)
		upstreamMesoNodeID := upstreamMesoNode.ID
		for segmentIdx := 0; segmentIdx < segmentsToCut; segmentIdx++ {
			// Prepare mesoscopic node
			if targetMacroNode.IsCentroid() && segmentIdx == segmentsToCut-1 {
				return nil, nil, errors.Wrap(ErrNotImplementedYet, "Prepare downstream mesoscopic node from centroid")
			} else {
				expNodesNum, ok := expandedMesoNodes[macroLinkProcess.targetMacroNodeID]
				if !ok {
					expandedMesoNodes[macroLinkProcess.targetMacroNodeID] = 0
				}
				expandedMesoNodes[macroLinkProcess.targetMacroNodeID] += 1
				macroNodeID := gmns.NodeID(-1)
				macroLinkID := macroLinkProcess.id
				zoneID := gmns.NodeID(-1)
				activityLinkType := types.LINK_UNDEFINED
				if segmentIdx == segmentsToCut-1 {
					macroNodeID = macroLinkProcess.targetMacroNodeID
					macroLinkID = gmns.LinkID(-1)
					zoneID = targetMacroNode.Zone()
					activityLinkType = targetMacroNode.ActivityLinkType()
				}
				downstreamMesoNode = meso.NewNodeFrom(
					macroLinkProcess.targetMacroNodeID*100+gmns.NodeID(expNodesNum),
					meso.WithPointGeom(macroLinkProcess.offsetGeomCut[segmentIdx][len(macroLinkProcess.offsetGeomCut[segmentIdx])-1]), // No explicit copy or clone method since Point is not slice, but array
					meso.WithPointEuclideanGeom(macroLinkProcess.offsetGeomEuclideanCut[segmentIdx][len(macroLinkProcess.offsetGeomEuclideanCut[segmentIdx])-1]),
					meso.WithPointMacroNode(macroNodeID),
					meso.WithPointMacroLink(macroLinkID),
					meso.WithMacroZone(zoneID),
					meso.WithActivityLinkType(activityLinkType),
					meso.WithBoundaryType(types.BOUNDARY_NONE),
				)
				collectedMesoNodes[downstreamMesoNode.ID] = downstreamMesoNode
			}

			mesoLink := meso.NewLinkFrom(
				lastMesoLinkID,
				upstreamMesoNodeID,
				downstreamMesoNode.ID,
				meso.WithLanesNum(macroLinkProcess.lanesInfoCut.LanesList[segmentIdx]),
				meso.WithLanesChange(macroLinkProcess.lanesInfoCut.LanesChange[segmentIdx]),
				meso.WithLineGeom(macroLinkProcess.offsetGeomCut[segmentIdx].Clone()),
				meso.WithLineEuclideanGeom(macroLinkProcess.offsetGeomEuclideanCut[segmentIdx].Clone()),
				meso.WithLineMacroLink(macroLinkProcess.id),
				meso.WithSegmentIdx(segmentIdx),
				meso.WithMovement(-1),
				meso.WithLineMacroNode(-1),
				meso.WithLengthMeters(geo.LengthHaversine(macroLinkProcess.offsetGeomCut[segmentIdx])),
			)
			meso.WithOutcomingLinks(lastMesoLinkID)(collectedMesoNodes[upstreamMesoNodeID])
			meso.WithIncomingLinks(lastMesoLinkID)(collectedMesoNodes[downstreamMesoNode.ID])

			// Prepare mesoscopic link
			collectedMesoLinks[mesoLink.ID] = mesoLink
			lastMesoLinkID += 1
			upstreamMesoNodeID = downstreamMesoNode.ID // This must be done since current upstream node is downstream node for next segment
		}
	}
	return collectedMesoNodes, collectedMesoLinks, nil
}

func connectMesoscopicLinks(
	mesoLinks map[gmns.LinkID]*meso.Link,
	mesoNodes map[gmns.NodeID]*meso.Node,
	macroNodes map[gmns.NodeID]*macro.Node,
	macroLinks map[gmns.LinkID]*macro.Link,
	macroNodesMovements map[gmns.NodeID][]*movement.Movement,
	macroNodesNeedMovement map[gmns.NodeID]bool,
) error {
	lastMesoLinkID := gmns.LinkID(0)

	// Find max ID (for further links creating)
	for _, mesoLink := range mesoLinks {
		if mesoLink.ID > lastMesoLinkID {
			lastMesoLinkID = mesoLink.ID
		}
	}
	lastMesoLinkID++

	// Collect mesoscopic links for parent macroscopic links
	macroLinkMesoLinks := make(map[gmns.LinkID][]*meso.Link)
	for i := range mesoLinks {
		macroLinkID := mesoLinks[i].MacroLinkID()
		if _, ok := macroLinkMesoLinks[macroLinkID]; !ok {
			macroLinkMesoLinks[macroLinkID] = make([]*meso.Link, 0, 1)
		}
		macroLinkMesoLinks[macroLinkID] = append(macroLinkMesoLinks[macroLinkID], mesoLinks[i])
	}
	for i := range macroLinkMesoLinks {
		macroLinkData := macroLinkMesoLinks[i]
		/* Sort mesoscopic links by its segment number in parent macroscopic link */
		// @todo: We can achieve better perfomance if does sort during populating data
		sort.Slice(macroLinkData, func(i, j int) bool {
			return macroLinkData[i].SegmentIdx() < macroLinkData[j].SegmentIdx()
		})
	}

	collectedMesoLinks := make(map[gmns.LinkID]*meso.Link)
	// Start main loop for finding connections between mesoscopic links
	for macroNodeID := range macroNodes {
		// macroNode := macroNodes[macroNodeID]
		macroNodeMvmts, ok := macroNodesMovements[macroNodeID]
		if !ok {
			continue
		}
		for j := range macroNodeMvmts {
			mvmt := macroNodeMvmts[j]
			incomingMacroLink, ok := macroLinks[mvmt.IncomeMacroLinkID]
			if !ok {
				return errors.Wrapf(macro.ErrLinkNotFound, "Can't find macro link for further connection: %d", mvmt.IncomeMacroLinkID)
			}
			outcomingMacroLink, ok := macroLinks[mvmt.OutcomeMacroLinkID]
			if !ok {
				return errors.Wrapf(macro.ErrLinkNotFound, "Can't find macro link for further connection: %d", mvmt.OutcomeMacroLinkID)
			}

			incomingMesolinks := macroLinkMesoLinks[incomingMacroLink.ID]
			if len(incomingMesolinks) == 0 {
				panic("No mesoscopic links for incoming macro link")
			}
			outcomingMesolinks := macroLinkMesoLinks[outcomingMacroLink.ID]
			if len(outcomingMesolinks) == 0 {
				panic("No mesoscopic links for outcoming macro link")
			}

			incomigMesoLink := incomingMesolinks[len(incomingMesolinks)-1]
			incomigMesoLinkGeom := incomigMesoLink.Geom()
			incomigMesoLinkGeomEuclidean := incomigMesoLink.GeomEuclidean()
			outcomigMesoLink := outcomingMesolinks[0]
			outcomigMesoLinkGeom := outcomigMesoLink.Geom()
			outcomigMesoLinkGeomEuclidean := outcomigMesoLink.GeomEuclidean()

			geom := orb.LineString{incomigMesoLinkGeom[len(incomigMesoLinkGeom)-1], outcomigMesoLinkGeom[0]}
			geomEuclidean := orb.LineString{incomigMesoLinkGeomEuclidean[len(incomigMesoLinkGeomEuclidean)-1], outcomigMesoLinkGeomEuclidean[0]}
			if macroNodesNeedMovement[macroNodeID] {
				sourceMesoNodeID := incomigMesoLink.TargetNodeID()
				targetMesoNodeID := outcomigMesoLink.SourceNodeID()
				mesoLink := meso.NewLinkFrom(
					lastMesoLinkID,
					sourceMesoNodeID,
					targetMesoNodeID,
					meso.WithLanesNum(mvmt.LanesNum()),
					meso.WithLineGeom(geom),
					meso.WithLineEuclideanGeom(geomEuclidean),
					meso.WithLineMacroLink(-1),
					meso.Connection(true),
					meso.WithMovement(mvmt.ID),
					meso.WithLineMacroNode(macroNodeID),
					meso.WithLengthMeters(geo.LengthHaversine(geom)),
					/* mmvmt properties: todo */
				)
				meso.WithOutcomingLinks(lastMesoLinkID)(mesoNodes[sourceMesoNodeID])
				meso.WithIncomingLinks(lastMesoLinkID)(mesoNodes[targetMesoNodeID])
				// Prepare mesoscopic link
				collectedMesoLinks[mesoLink.ID] = mesoLink
				lastMesoLinkID += 1
			} else {
				panic("@todo: delete redundant node")
			}
		}
	}
	for i := range collectedMesoLinks {
		mesoLinks[collectedMesoLinks[i].ID] = collectedMesoLinks[i]
	}
	return nil
}
