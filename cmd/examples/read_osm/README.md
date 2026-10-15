# Read OSM example

[main.go](main.go) shows how to use osm2gmns as a Go package: read OSM, generate macro, movements, meso and micro, then export GMNS datasets.

Place an OSM XML file named `sample.osm` in the repository root and run from that directory:

```shell
go run ./cmd/examples/read_osm
```

The example writes the results to `./output`. To use another `.osm` or `.osm.pbf` file, change the path passed to `NewParser` in `main.go`.
