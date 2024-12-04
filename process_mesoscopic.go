package osm2gmns

import (
	"math"
	"time"

	"github.com/LdDl/osm2gmns/geomath"
	"github.com/LdDl/osm2gmns/gmns"
	"github.com/LdDl/osm2gmns/macro"
	"github.com/LdDl/osm2gmns/meso"
	"github.com/LdDl/osm2gmns/movement"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type macroLinkProcessing struct {
	needsOffset         bool
	id                  gmns.LinkID
	offsetGeomEuclidean orb.LineString
	offsetGeom          orb.LineString
	lanesInfo           macro.LanesInfo
	lengthMetersOffset  float64
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
	needToObserve := make(map[gmns.LinkID]*macroLinkProcessing)
	macroLinks := macroLinksToSlice(macroNet.Links)
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
		macroLinkProcessing := needToObserve[macroLinkID]
		macroLink, ok := macroNet.Links[macroLinkID]
		if !ok {
			return nil, errors.Wrapf(macro.ErrLinkNotFound, "Offset Link ID: %d", macroLinkID)
		}
		if !macroLinkProcessing.needsOffset {
			macroLinkProcessing.offsetGeomEuclidean = macroLink.GeomEuclidean().Clone()
			macroLinkProcessing.offsetGeom = macroLink.Geom().Clone()
			continue
		}
		offsetDistance := 2 * (float64(macroLink.MaxLanes())/2 + 0.5) * macro.LANE_WIDTH
		macroLinkProcessing.offsetGeomEuclidean = geomath.OffsetCurve(macroLink.GeomEuclidean(), -offsetDistance)
		macroLinkProcessing.offsetGeom = geomath.LineToSpherical(macroLinkProcessing.offsetGeomEuclidean)
	}

	// Update breakpoints since geometry has changed
	for macroLinkID := range needToObserve {
		macroLinkProcessing := needToObserve[macroLinkID]
		// Re-calcuate length for offset geometry and round to 2 decimal places
		macroLinkProcessing.lengthMetersOffset = math.Round(geo.LengthHaversign(macroLinkProcessing.offsetGeom)*100.0) / 100.0
		macroLink, ok := macroNet.Links[macroLinkID]
		if !ok {
			return nil, errors.Wrapf(macro.ErrLinkNotFound, "Offset Link ID: %d", macroLinkID)
		}
		for i, item := range macroLinkProcessing.lanesInfo.LanesChangePoints {
			macroLinkProcessing.lanesInfo.LanesChangePoints[i] = (item / macroLink.LengthMeters()) * macroLinkProcessing.lengthMetersOffset
		}
	}
	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Msg("Process movements")
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
