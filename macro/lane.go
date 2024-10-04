package macro

import (
	"sort"
)

const (
	resolution = 5.0
)

type LanesInfo struct {
	LanesList         []int
	LanesChange       [][2]int
	LanesChangePoints []float64
}

func NewLanesInfo(link *Link) *LanesInfo {
	lanesInfo := &LanesInfo{
		LanesList:         make([]int, 0),
		LanesChange:       make([][2]int, 0),
		LanesChangePoints: make([]float64, 0),
	}
	lanesChangePointsTemp := []float64{0.0, link.lengthMeters}
	if link.lengthMeters < resolution {
		lanesInfo.LanesChangePoints = []float64{0.0, link.lengthMeters}
	} else {
		for len(lanesChangePointsTemp) != 0 {
			target := lanesChangePointsTemp[0]
			remove := make(map[int]struct{})
			for idx, point := range lanesChangePointsTemp {
				if target-resolution <= point && point <= target+resolution {
					remove[idx] = struct{}{}
				}
			}
			lanesInfo.LanesChangePoints = append(lanesInfo.LanesChangePoints, target)
			for idx := range remove {
				lanesChangePointsTemp = append(lanesChangePointsTemp[:idx], lanesChangePointsTemp[idx+1:]...)
			}
		}
		sort.Float64s(lanesInfo.LanesChangePoints)
	}
	for i := 0; i < len(lanesInfo.LanesChangePoints)-1; i++ {
		lanesInfo.LanesList = append(lanesInfo.LanesList, link.lanesNum)
		lanesInfo.LanesChange = append(lanesInfo.LanesChange, [2]int{0.0, 0.0})
	}
	return lanesInfo
}
