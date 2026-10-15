package csvvalue

import (
	"strconv"

	"github.com/LdDl/go-gmns/gmns"
)

func Float(value float64, present bool) string {
	if !present {
		return ""
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func Int(value int, present bool) string {
	if !present {
		return ""
	}
	return strconv.Itoa(value)
}

func NodeID(value gmns.NodeID, present bool) string {
	if !present {
		return ""
	}
	return value.String()
}
