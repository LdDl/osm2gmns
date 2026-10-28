package osmmacro

import (
	"strconv"
	"strings"

	"github.com/paulmach/osm"
)

func wayDiagnostics(tags osm.Tags) []SourceDiagnostic {
	result := make([]SourceDiagnostic, 0)
	seen := make(map[string]bool)
	values := make(map[string]string)
	for _, tag := range tags {
		if seen[tag.Key] {
			result = append(result, SourceDiagnostic{
				Code:    "repeated_tag",
				Message: "repeated " + tag.Key + " evidence requires resolution",
			})
		}
		seen[tag.Key] = true
		values[tag.Key] = tag.Value
		if tag.Key == "incline" || strings.HasPrefix(tag.Key, "incline:") {
			result = append(result, SourceDiagnostic{
				Code:    "incline_not_mapped",
				Message: "incline retained without assuming units, direction, span or a numerical grade",
			})
		}
		if tag.Key == "ele" || strings.HasPrefix(tag.Key, "ele:") {
			result = append(result, SourceDiagnostic{
				Code:    "way_elevation_not_mapped",
				Message: "way elevation retained without assigning it to every road point",
			})
		}
	}
	bridge := values["bridge"]
	tunnel := values["tunnel"]
	if bridge != "" && bridge != "no" || tunnel != "" && tunnel != "no" {
		result = append(result, SourceDiagnostic{
			Code:    "vertical_profile_unknown",
			Message: "structure tags do not establish metric clearance or a vertical profile",
		})
	}
	if bridge != "" && bridge != "no" && tunnel != "" && tunnel != "no" {
		result = append(result, SourceDiagnostic{
			Code:    "conflicting_structure",
			Message: "bridge and tunnel evidence require review",
		})
	}
	if seen["bridge"] && bridge != "yes" && bridge != "no" {
		result = append(result, SourceDiagnostic{
			Code:    "bridge_classification",
			Message: "bridge value retained for explicit classification review",
		})
	}
	if seen["tunnel"] && tunnel != "yes" && tunnel != "no" {
		result = append(result, SourceDiagnostic{
			Code:    "tunnel_classification",
			Message: "tunnel value retained; building passages and other values are not assumed underground",
		})
	}
	if seen["covered"] {
		result = append(result, SourceDiagnostic{
			Code:    "covered_not_tunnel",
			Message: "covered evidence does not establish an underground road",
		})
	}
	if seen["layer"] && bridge == "" && tunnel == "" {
		result = append(result, SourceDiagnostic{
			Code:    "relative_layer_only",
			Message: "layer is relative crossing order, not metric elevation or tunnel classification",
		})
	}
	if seen["layer"] {
		_, err := strconv.Atoi(values["layer"])
		if err != nil {
			result = append(result, SourceDiagnostic{
				Code:    "unsupported_layer",
				Message: "layer is not a single integer; retained without interpretation",
			})
		}
	}
	if values["oneway"] == "-1" {
		result = append(result, SourceDiagnostic{
			Code:    "reverse_oneway_unsupported",
			Message: "existing converter does not reverse oneway=-1; source mapping describes the generated direction, traffic direction requires review",
		})
	}
	return result
}
