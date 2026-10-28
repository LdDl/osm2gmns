# OSM source evidence

The optional `osm_source` object in `provenance_macro.json` retains acquisition evidence outside GMNS resources. It does not change standard columns, network connectivity or supported GMNS import. The CLI includes it by default; `-source-metadata=false` keeps the original provenance shape for strict readers. This flag does not suppress standard node elevations or warnings.

## Contents

| Collection | Meaning |
|---|---|
| `ways` | Exact OSM way ID, original ordered `node_refs` and retained tags |
| `links` | GMNS link ID, OSM way ID, inclusive zero-based `start_index`/`end_index`, and `direction` |
| `nodes` | Tagged source points with original coordinates and tags; `node_id` only when an actual GMNS node exists |
| `diagnostics` | Source kind/ID, diagnostic code and explanation |

All IDs in this extension are decimal strings. Tags are ordered `{key, value}` records, retaining empty values and duplicates. Retained keys are `bridge`, `tunnel`, `covered`, `layer`, `level`, `oneway`, `ele`, `ele:*`, `source:ele`, `incline` and `incline:*`. Only ways represented in the converted macro network and their tagged source nodes are included. This is not an archive of excluded OSM objects.

`direction=1` follows the source way's node order; `-1` reverses the selected span. Indices refer to occurrences, so repeated node IDs and closed ways remain unambiguous. Each converted link has one exact source span. Shared geometry does not change this mapping; standard `dir_flag` describes orientation relative to a geometry record and is a separate concept. No source matching by names or proximity is performed.

An existing conversion limitation leaves `oneway=-1` in source order. Such ways receive `reverse_oneway_unsupported`; the span describes actual generated geometry, not certified traffic permissions. Generated movements also remain subject to the converter's turn-restriction limitations.

## Elevation and classification

One finite decimal node `ele`, optionally suffixed by ` m`, is interpreted in meters above EGM96 according to the OSM convention. An explicit `ele:datum` other than `EGM96`, repeated `ele`, unsupported units or nonnumeric values prevent standard altitude assignment. Alternative `ele:*` values remain evidence; no datum conversion or measurement verification is performed. Missing, zero and negative heights remain distinct.

Only actual macro nodes receive `z_coord`. Elevations at intermediate shape points remain attached to their source node IDs; conversion does not introduce topology to store them. Way-level elevations and all incline values remain evidence because their scope, reference and interpretation need an explicit policy. No grade or complete profile is inferred from them.

`layer` is relative crossing order, never meters or a global floor number. Negative layer alone does not classify a tunnel. `tunnel=building_passage`, `covered` and unsupported classifications receive diagnostics rather than being converted into underground roads. Contradictory or repeated source tags are retained for review. Warnings are printed unless `-suppress-warnings=true`; diagnostics remain in the metadata when enabled.

## API and compatibility

`GenerateMacroscopicWithEvidence` returns the network and an independent `osmmacro.SourceEvidence`. After normal preparation, call `evidence.Attach(net, &dataset)` on the result of `osmmacro.BuildDataset`. Capturing happens during generation. Attachment only serializes the snapshot and checks network identities, endpoints and geometry; stale mappings are rejected. Existing generation and export entry points remain available without evidence.

The `osm_source` object declares `format="osm2gmns-osm-source"` and `version=1`. Readers should ignore unknown fields within a supported extension and skip unsupported extension versions while continuing to process standard GMNS records.

`csvimport.ReadMacro` ignores provenance entirely. For consumers that reject additional provenance fields, use `-source-metadata=false` to omit `osm_source`.
