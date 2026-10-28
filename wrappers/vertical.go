package wrappers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/paulmach/osm"
)

var elevationNumber = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?(?: m)?$`)

var tagsSource map[string]struct{} = map[string]struct{}{
	"bridge":     {},
	"tunnel":     {},
	"covered":    {},
	"layer":      {},
	"level":      {},
	"oneway":     {},
	"ele":        {},
	"source:ele": {},
	"incline":    {},
}

// SourceTags preserves source spelling, empty values and repeated keys in source order.
func SourceTags(tags osm.Tags) osm.Tags {
	result := make(osm.Tags, 0)
	for _, tag := range tags {
		if _, ok := tagsSource[tag.Key]; ok || strings.HasPrefix(tag.Key, "ele:") || strings.HasPrefix(tag.Key, "incline:") {
			result = append(result, tag)
		}
	}
	return result
}

// SourceTags returns an independent copy of retained acquisition evidence.
func (tags *WayTags) SourceTags() osm.Tags {
	return append(osm.Tags(nil), tags.sourceTags...)
}

// NodeElevation interprets one ele value as meters above EGM96 under the OSM convention.
// It performs no datum conversion; alternate datums remain source evidence only.
func NodeElevation(tags osm.Tags) (float64, bool, error) {
	value := ""
	count := 0
	for _, tag := range tags {
		if tag.Key == "ele" {
			value = tag.Value
			count++
		}
	}
	if count == 0 {
		return 0, false, nil
	}
	if count != 1 {
		return 0, false, fmt.Errorf("repeated ele values require resolution")
	}
	for _, tag := range tags {
		if tag.Key == "ele:datum" && tag.Value != "EGM96" {
			return 0, false, fmt.Errorf("unsupported explicit elevation datum %q", tag.Value)
		}
	}
	value = strings.TrimSpace(value)
	if !elevationNumber.MatchString(value) {
		return 0, false, fmt.Errorf("ele must be a decimal number in meters, optionally followed by ' m'")
	}
	value = strings.TrimSuffix(value, " m")
	altitude, err := strconv.ParseFloat(value, 64)
	if err != nil || !gmns.Finite(altitude) {
		return 0, false, fmt.Errorf("ele must be finite and representable")
	}
	return altitude, true, nil
}
