## osm2gmns
Just port of https://github.com/jiawlu/OSM2GMNS in Golang.

This tool allows to prepare routable graph of a given OSM road network with three levels of detalization:
1) Macroscopic + movements layer;
    * Images: W.I.P.
    * Data and fields description: W.I.P.

2) Mesoscopic;
    * Images: W.I.P.
    * Data and fields description: W.I.P.

3) Microscopic.
    * Images: W.I.P.
    * Data and fields description: W.I.P.

Stage: **W.I.P.**

* How to use as binary executable
W.I.P.

* How to use as a package
  - Get latest version of the package
    ```shell
    go get github.com/LdDl/osm2gmns@latest 
    ```
  - Prepare *.osm (XML) or *.pbf (Protobuf) file with OSM data 
  - Code:
    ```go
    package main

    import (
	    "fmt"

	    "github.com/LdDl/go-gmns/generators"
	    "github.com/LdDl/go-gmns/gmns/types"
	    "github.com/LdDl/osm2gmns"
    )

    func main() {
	    // Explicitly set global variables for logging (those are defaults actually)
	    osm2gmns.VERBOSE = true
	    osm2gmns.SUPPRESS_WARNINGS = false

	    parser := osm2gmns.NewParser(
		    "./sample.osm",
		    osm2gmns.WithPreparePOI(false),
		    osm2gmns.WithStrictMode(false),
		    osm2gmns.WithVerbose(true),
		    osm2gmns.WithAllowedAgentTypes(types.AGENT_TYPES_DEFAULT),
	    )
	    fmt.Println(parser)

	    osmData, err := parser.ReadOSM()
	    if err != nil {
		    fmt.Println(err)
		    return
	    }
	    macroNet, err := osm2gmns.GenerateMacroscopic(osmData, false)
	    if err != nil {
		    fmt.Println(err)
		    return
	    }
	    fmt.Println("macro nodes num", len(macroNet.Nodes))
	    fmt.Println("macro links num", len(macroNet.Links))

	    movements, err := generators.GenerateMovements(macroNet)
	    if err != nil {
		    fmt.Println(err)
		    return
	    }
	    fmt.Println("movements_num", len(movements))

	    mesoNet, err := generators.GenerateMesoscopic(macroNet, movements)
	    if err != nil {
		    fmt.Println(err)
		    return
	    }

	    fmt.Println("meso nodes num", len(mesoNet.Nodes))
	    fmt.Println("meso links num", len(mesoNet.Links))
    }
    ```

* Test:
  ```shell
  # Load some OSM file to root of folder
  # Call this file sample.osm
  # Run:
  go test -timeout 30s -run '^TestParser$' ./*.go
  # After you will see some files in test_data folder
  ```
