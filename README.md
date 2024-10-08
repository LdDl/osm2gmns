## osm2gmns
Just port of https://github.com/jiawlu/OSM2GMNS in Golang

Stage: **W.I.P.**

Current test:
```shell
# Load some OSM file to root of folder
# Call this file sample.osm
# Run:
go test -timeout 30s -run '^TestParser$' ./*.go
# After you will see some files in test_data folder
```