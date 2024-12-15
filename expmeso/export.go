package expmeso

import (
	"fmt"
	"strings"

	"github.com/LdDl/go-gmns/meso"
)

func ExportToCSV(macroNet *meso.Net, fname string) error {
	fnameParts := strings.Split(fname, ".csv")
	fnameNodes := fmt.Sprintf(fnameParts[0] + "_meso_nodes.csv")
	fnameLinks := fmt.Sprintf(fnameParts[0] + "_meso_links.csv")
	_ = fnameNodes
	_ = fnameLinks
	panic("@todo")
	return nil
}
