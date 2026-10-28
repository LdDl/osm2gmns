# GMNS export

Each network is a separate dataset in a flat directory. `datapackage_macro.json`, `datapackage_meso.json` and `datapackage_micro.json` map standard logical resource names to files such as `node_macro.csv`. Foreign keys resolve inside that dataset. The suffix convention belongs to this converter; GMNS does not prescribe network folders.

## Supported records

| Table | Source |
|---|---|
| `config` | Dataset configuration, metric units, WGS84, WKT and ID type |
| `node`, `link` | Prepared network records |
| `movement` | Movements owned by the network |
| `geometry`, `zone` | Prepared geometry and zone catalogs |
| `use_definition`, `use_group` | Use definitions and supplied groups |
| `lane`, `segment`, `segment_lane` | Supplied detailed records |

Config, node and link files are always written. Empty optional tables are omitted. Supplied supported attributes and records are retained; missing values stay blank, supplied zeroes and negative lane numbers stay numeric. IDs retain their configured kind and spelling, including string `"001"`. Config declares the ID type; CSV itself cannot distinguish numeric and string tokens. Literal `NaN` is rejected because the standard schemas reserve it as a missing-value marker.

The standard schemas are linked from each descriptor. OSM origin IDs, source network references, calculated indices and generator codes belong to `provenance_<level>.json`, outside the standard resources. Standard parent references always target the same dataset.

## Generation and ownership

The pipeline is OSM parsing, macro generation, movement generation, meso generation, micro generation, explicit preparation, then serialization. Run all required generators before mapping built-in agents to standard uses. The [Go example](../cmd/examples/read_osm/README.md) shows this order.

Macro movements reference macro nodes and links. Meso represents road segments and junction connectors; micro represents cells and lane connections. These generators do not populate their own movement collections. Exporters support supplied movements with valid references to their owning network. A source movement need not produce exactly one connector; source relationships are retained as provenance.

`-networks movement` selects the macro dataset, including its nodes and links. Selecting macro also includes its movements. Selecting only meso or micro does not export prerequisite levels.

Movement generation builds candidates from topology and geometry. OSM turn restriction relations are not processed, so generated movements are not proof that a turn is legally permitted. Cut distances, offsets and connector shortcuts are generator choices, not GMNS requirements. Generated U-turn codes that cannot be represented by standard `mvmt_code` are retained in provenance; the standard movement keeps its `type` and an empty code. Invalid supplied codes produce an export error.

Generators require prepared lane counts and lane data for the operations that consume them. They reject supplied detailed lane tables, arbitrary authored use mappings and nonzero road tolls when their transformation is unsupported. Exporting an already prepared network can preserve these records without running a generator. Calculated lane indices do not fabricate standard lane identities.

## Uses and units

`preparation.DefaultUseOptions()` uses named constants: auto occupancy 1 / PCE 1, bike occupancy 1 / PCE 0.5, walk occupancy 1 / PCE 0. These are modeling assumptions, not observations extracted from OSM. Change the options before preparation or use the CLI flags listed in the [README](../README.md#cli-options). Preparation keeps existing definitions and nonempty authored `allowed_uses`.

The working model uses metric units: meters, km/h and seconds, with WGS84 coordinates. Link capacity is PCE/hour/lane; movement capacity is PCE/hour for the whole movement. Missing penalties and capacities are not inferred. Imperial conversion and other coordinate systems are outside the current export scope.

## Geometry

`-geometry inline` writes each link's travel-order WKT with `dir_flag=1` for directed links. `-geometry shared` writes macro geometry references with `dir_flag=1` or `-1` relative to the stored shape. Meso and micro always use inline geometry. Undirected inline links use `dir_flag=0`.

Geometry IDs and canonical shapes must already exist before shared export. Serialization does not allocate IDs or repair missing geometry. Prepared geometry catalogs are retained in both modes, even when inline links no longer reference them; provenance retains the original geometry reference and direction. Consequently, inline mode can contain both per-link shapes and a geometry catalog. Shared mode avoids repeating the referenced shapes in links.

## Source evidence and heights

`-source-metadata` includes captured OSM tags and exact source spans in macro provenance. Use `-source-metadata=false` for readers that require the original metadata shape. Standard tables remain independent of this optional evidence. See [source evidence](source-evidence.md) for the supported tags and mapping.

Supported node `ele` values populate `z_coord` in meters under the OSM EGM96 convention. Invalid or unknown values remain absent. Intermediate shape-point elevations remain evidence because line geometry is two-dimensional. Structure tags never generate heights, and `incline` is retained without assigning a numerical `grade`.

## Export API and writes

CSV uses `;` by default. Set `-delimiter ","` in the CLI or pass `csvexport.WithDelimiter(',')` to `ExportToCSV`, `ExportToCSVWithGeometry`, `WriteDatasets` or `WriteTable`. The separator must be one valid UTF-8 character; double quotes, CR, LF, NUL and the Unicode replacement character are rejected. Every datapackage resource declares the chosen separator. Comma-separated lists inside `allowed_uses` and `use_group.uses` keep their standard meaning regardless of the CSV separator.

`osmmacro`, `osmmeso` and `osmmicro` provide `BuildDataset` and `ExportToCSV`. `BuildDataset` validates the prepared network and builds a serialization snapshot without changing it. Use `csvexport.WriteDatasets` to write several snapshots together. `osmmovement.ExportToCSV` writes a standalone movement table; a full network export also validates configuration and local references.

The writer stages every selected dataset before replacing managed files, removes stale optional tables for those levels, and leaves unrelated files and unselected levels untouched. Ordinary replacement errors trigger rollback; if rollback fails, the error identifies retained backups. The multi-file operation is not atomic against process termination, power loss or concurrent access. Avoid concurrent writers to one output directory.

Signal timing, time-of-day, location and curb tables are outside the supported scope. May be I will add them in the future.
