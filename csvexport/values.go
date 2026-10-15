package csvexport

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
)

var movementCodePattern = regexp.MustCompile(`^[NSEW][EWB][RLT]\d?$`)

func ValidMovementCode(code string) bool {
	return code == "" || movementCodePattern.MatchString(code)
}

func Number(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func LinkID(value gmns.LinkID, present bool) string {
	if !present {
		return ""
	}
	return value.String()
}

func LaneID(value gmns.LaneID, present bool) string {
	if !present {
		return ""
	}
	return value.String()
}

func Line(shape orb.LineString) string {
	if len(shape) == 0 {
		return ""
	}
	return wkt.MarshalString(shape)
}

func Boundary(shape orb.MultiPolygon) string {
	if len(shape) == 0 {
		return ""
	}
	return wkt.MarshalString(shape)
}

// GeometryCells selects a representation without modifying or reversing either shape.
func GeometryCells(id gmns.GeometryID, hasID bool, direction gmns.DirectionFlag, hasDirection bool, shape orb.LineString, directed, shared bool) (string, string, string, error) {
	if shared {
		if !hasID || !hasDirection {
			return "", "", "", fmt.Errorf("shared geometry requires a prepared geometry_id and dir_flag")
		}
		if !directed {
			return id.String(), "", "0", nil
		}
		return id.String(), "", strconv.Itoa(int(direction)), nil
	}
	if len(shape) == 0 {
		if hasID {
			return "", "", "", fmt.Errorf("inline geometry requires a prepared shape")
		}
		return "", "", "", nil
	}
	// Inline shapes in these models are already ordered from source to target.
	if !directed {
		return "", Line(shape), "0", nil
	}
	return "", Line(shape), "1", nil
}
