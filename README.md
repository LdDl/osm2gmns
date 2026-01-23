## osm2gmns

Go port of [OSM2GMNS](https://github.com/jiawlu/OSM2GMNS) - convert OpenStreetMap data to GMNS format.

This tool prepares a routable graph from OSM road network with three levels of detail:
1. **Macroscopic** + movements layer
2. **Mesoscopic** (lane-level)
3. **Microscopic** (cell-based)

See [go-gmns](https://github.com/LdDl/go-gmns) for data format documentation.

## Installation

### Option 1: Go install

```shell
go install github.com/LdDl/osm2gmns/cmd/osm2gmns@latest
```

### Option 2: Download binary

Download the latest release from [GitHub Releases](https://github.com/LdDl/osm2gmns/releases).

**Linux/macOS:**
```shell
tar -xzf osm2gmns-*-linux-amd64.tar.gz
chmod +x osm2gmns-linux-amd64/osm2gmns
sudo mv osm2gmns-linux-amd64/osm2gmns /usr/local/bin/
```

**Windows:**
Extract the zip file and add the folder to your PATH.

### Option 3: Docker

```shell
docker pull dimahkiin/osm2gmns:latest
```

## CLI Usage

```shell
# Basic usage - generate all network levels
osm2gmns -input map.osm -output ./results

# Generate only macro and meso networks
osm2gmns -input map.osm.pbf -networks macro,meso -output ./results

# Specify agent types
osm2gmns -input map.osm -agents auto,bike -output ./results

# Quiet mode
osm2gmns -input map.osm -output ./results -verbose=false

# Show version
osm2gmns -version

# Show help
osm2gmns -help
```

### CLI Options

| Flag | Default | Description |
|------|---------|-------------|
| `-input` | (required) | Input OSM file path (.osm or .osm.pbf) |
| `-output` | `./output` | Output directory for CSV files |
| `-networks` | `macro,movement,meso,micro` | Network levels to generate |
| `-agents` | `auto,bike,walk` | Allowed agent types |
| `-verbose` | `true` | Enable verbose output |
| `-suppress-warnings` | `false` | Suppress warning messages |
| `-strict` | `false` | Enable strict mode |
| `-poi` | `false` | Prepare POI data |
| `-version` | - | Show version information |

### Docker Usage

```shell
# Process local OSM file
docker run -v $(pwd):/data dimahkiin/osm2gmns -input /data/map.osm -output /data/results

# With specific options
docker run -v $(pwd):/data dimahkiin/osm2gmns \
    -input /data/map.osm.pbf \
    -output /data/results \
    -networks macro,meso \
    -agents auto
```

## Data Preparation

For large OSM files, consider preprocessing with [osmconvert](https://wiki.openstreetmap.org/wiki/Osmconvert) to reduce size:

**Install osmconvert (Linux):**
```shell
wget -O - http://m.m.i24.cc/osmconvert.c | cc -x c - -lz -O3 -o osmconvert
sudo mv osmconvert /usr/local/bin/
```

**Convert and filter:**
```shell
# Convert XML to PBF (smaller, faster to parse)
osmconvert map.osm --out-pbf -o=map.osm.pbf

# Remove metadata to reduce size
osmconvert map.osm --drop-author --drop-version --out-pbf -o=map.osm.pbf
```

## Library Usage

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

	    microNet, err := generators.GenerateMicroscopic(macroNet, mesoNet, movements)
	    if err != nil {
		    fmt.Println(err)
		    return
	    }
	    fmt.Println("micro nodes num", len(microNet.Nodes))
	    fmt.Println("micro links num", len(microNet.Links))
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
