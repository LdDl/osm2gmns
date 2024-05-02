package macro

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

func (net *Net) exportNodesToCSV(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "osm_node_id", "control_type", "boundary_type", "activity_type", "activity_link_type", "zone_id", "intersection_id", "poi_id", "osm_highway", "name", "longitude", "latitude"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for _, node := range net.Nodes {
		err = writer.Write([]string{
			fmt.Sprintf("%d", node.ID),
			fmt.Sprintf("%d", node.osmNodeID),
			node.controlType.String(),
			node.boundaryType.String(),
			node.activityType.String(),
			node.activityLinkType.String(),
			fmt.Sprintf("%d", node.zoneID),
			fmt.Sprintf("%d", node.intersectionID),
			fmt.Sprintf("%d", node.poiID),
			node.osmHighway,
			node.name,
			fmt.Sprintf("%f", node.geom[0]),
			fmt.Sprintf("%f", node.geom[1]),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write node")
		}
	}
	return nil
}
