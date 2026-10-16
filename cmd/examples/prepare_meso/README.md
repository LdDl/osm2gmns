# Prepare meso from GMNS

This local runner reads a macro datapackage and writes macro/meso datasets into a new flat directory. It needs no OSM file. Run from the osm2gmns repository root.

```sh
go run ./cmd/examples/prepare_meso \
    -input /path/to/datapackage_macro.json \
    -output /path/to/new-result \
    -lane-layout uniform \
    -lane-connections ordered
```

Use `-lane-layout detailed` for supplied `lane`, `segment` and `segment_lane` tables. Uncovered intervals use the base link layout. `-lane-connections ordered` explicitly pairs equal-sized movement ranges from left to right; the default `single` accepts only one lane per end. Blank end-lane values mean a single lane. Preparation never chooses replacement lanes or invents missing movements.

Directed links require positive metric lengths and lane counts 1..99. `-default-lanes N` fills only absent counts and records assumptions in provenance. Supplied zeroes and negative counts remain errors. WGS84, WKT, inline/shared geometry and use definitions/groups are supported. Detailed preparation supports nonoverlapping intervals and edge lane additions/drops; unsupported details return record-specific errors.

The descriptor supplies resource paths and input delimiters; omitted delimiters mean comma. Output defaults to `;`, configurable with `-delimiter ","`. Output geometry is inline. Existing output directories are rejected. Source detailed records remain in macro tables; generated section associations and ordered lane pairs are written to provenance. Existing provenance is not imported.

See [meso preparation](https://github.com/LdDl/go-gmns/blob/v0.2.1/docs/meso-preparation.md) for inheritance rules and supported detail. The installed osm2gmns executable is unaffected.
