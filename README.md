## osm2gmns

Go port of [OSM2GMNS](https://github.com/jiawlu/OSM2GMNS) - convert OpenStreetMap data to GMNS format.

This tool prepares a routable graph from OSM road network with three levels of detail:

1. **Macroscopic** + movements layer
2. **Mesoscopic** (road segments and junction connectors)
3. **Microscopic** (cell-based)

See [go-gmns](https://github.com/LdDl/go-gmns) for data format documentation.

## Contents

- [Installation](#installation)
- [CLI usage](#cli-usage)
- [Output](#output)
- [Data preparation](#data-preparation)
- [Library usage](#library-usage)

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
| `-delimiter` | `;` | CSV separator: one character, for example `-delimiter ","` |
| `-geometry` | `inline` | Macro geometry: `inline` in `link_macro.csv`, or `shared` references to `geometry_macro.csv` |
| `-networks` | `macro,movement,meso,micro` | Levels to export; `movement` selects its owning macro dataset |
| `-agents` | `auto,bike,walk` | Allowed agent types |
| `-verbose` | `true` | Enable verbose output |
| `-suppress-warnings` | `false` | Suppress warning messages |
| `-strict` | `false` | Enable strict mode |
| `-poi` | `false` | Prepare POI data |
| `-version` | - | Show version information |

`-geometry shared` requires macro export; meso and micro geometry stays inline. Macro includes its movements. Selecting only micro generates prerequisites internally and exports only micro. Write boolean flags as `-verbose=false`.

Modeling defaults for `use_definition` are named constants in `preparation` and can be overridden with these flags:

| Use | Occupancy flag (default) | PCE flag (default) |
|---|---|---|
| auto | `-auto-persons` (1) | `-auto-pce` (1) |
| bike | `-bike-persons` (1) | `-bike-pce` (0.5) |
| walk | `-walk-persons` (1) | `-walk-pce` (0) |

### Docker Usage

```shell
# Process local OSM file
docker run --rm -v "$(pwd):/data" dimahkiin/osm2gmns -input /data/map.osm -output /data/results

# With specific options
docker run --rm -v "$(pwd):/data" dimahkiin/osm2gmns \
    -input /data/map.osm.pbf \
    -output /data/results \
    -networks macro,meso \
    -agents auto
```

## Output

All files share one directory. Each selected level has an independent [GMNS](https://github.com/zephyr-data-specs/GMNS) dataset:

| Files | Contents |
|---|---|
| `node_<level>.csv`, `link_<level>.csv` | Network nodes and links |
| `config_<level>.csv` | Units, coordinate system and ID type |
| `movement_<level>.csv` | Movements owned by this network, when present |
| `<table>_<level>.csv` | Prepared geometries, zones, use definitions and other supported records |
| `datapackage_<level>.json` | Maps standard resource names to the suffixed files and schema references |
| `provenance_<level>.json` | OSM origins and generator metadata outside the standard tables |

`<level>` is `macro`, `meso` or `micro`. CSV uses `;` by default; `-delimiter ","` selects commas. Optional missing values are blank, and shapes use WKT. The working units are metric: meters, km/h and seconds, with WGS84 coordinates. See [export details](docs/export.md) for the supported scope.

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

```shell
go get github.com/LdDl/osm2gmns@latest
```

Examples:

- [Read OSM](cmd/examples/read_osm/README.md): generate networks, prepare datasets and export CSV.
- [Prepare meso from GMNS](cmd/examples/prepare_meso/README.md): read a saved macro datapackage and prepare meso without an OSM file.

See [export details](docs/export.md) for the supported records, ownership and modeling assumptions.
