package osm2gmns

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
)

func lineAsString(l orb.LineString) string {
	agg := []string{}
	for _, pt := range l {
		agg = append(agg, fmt.Sprintf("[%f, %f]", pt.X(), pt.Y()))
	}
	return "[" + strings.Join(agg, ",") + "]"
}

func TestOffset(t *testing.T) {
	line := orb.LineString{{10.0, 10.0}, {15.0, 10.0}, {18.0, 15.0}, {18.0, 20.0}, {15.0, 24.0}, {12.0, 24.0}, {10.0, 18.0}, {10.0, 15.0}, {13.0, 12.0}, {15.0, 16.0}}
	distance := 1.0

	leftL := lineAsString(offsetCurve(line, distance))
	rightL := lineAsString(offsetCurve(line, -distance))

	correctLeft := "[[10.000000, 11.000000],[14.433810, 11.000000],[17.000000, 15.276984],[17.000000, 19.666667],[14.500000, 23.000000],[12.720759, 23.000000],[11.000000, 17.837722],[11.000000, 15.414214],[12.726049, 13.688165],[14.105573, 16.447214]]"
	if leftL != correctLeft {
		t.Errorf("Left offset line should be '%s' but got '%s'", correctLeft, leftL)
	}
	correctRight := "[[10.000000, 9.000000],[15.566190, 9.000000],[19.000000, 14.723016],[19.000000, 20.333333],[15.500000, 25.000000],[11.279241, 25.000000],[9.000000, 18.162278],[9.000000, 14.585786],[13.273951, 10.311835],[15.894427, 15.552786]]"
	if rightL != correctRight {
		t.Errorf("Right offset line should be '%s' but got '%s'", correctRight, rightL)
	}
}

func findDist(p1, p2 orb.Point) float64 {
	return math.Sqrt(math.Pow(p2.X()-p1.X(), 2) + math.Pow(p2.Y()-p1.Y(), 2))
}

func rotateVector(vec orb.Point, angle float64) orb.Point {
	rad := deg2rad(angle)
	return orb.Point{
		vec[0]*math.Cos(rad) - vec[1]*math.Sin(rad),
		vec[0]*math.Sin(rad) + vec[1]*math.Cos(rad),
	}
}

const (
	d2r = math.Pi / 180.0
)

func deg2rad(deg float64) float64 {
	return deg * d2r
}

func TestLineSubstring(t *testing.T) {
	lineWKT := "LINESTRING (37.56319128200903 55.78357465483572, 37.565235359279626 55.78497472894253, 37.565822487858156 55.785421030200496, 37.567355545810614 55.784711836767826)"
	line, err := wkt.UnmarshalLineString(lineWKT)
	if err != nil {
		t.Error(err)
		return
	}
	newline := SubstringHaversine(line, 215, 278)
	newLineWKT := wkt.MarshalString(newline)
	correctLine := "LINESTRING(37.56536219999623 55.78507114703719,37.565822487858156 55.785421030200496,37.56600203415945 55.785337974305975)"
	if correctLine != newLineWKT {
		t.Errorf("Correct line should be '%s', but got '%s'", correctLine, newLineWKT)
	}
}
