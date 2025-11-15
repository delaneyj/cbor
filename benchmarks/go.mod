module github.com/delaneyj/cbor/benchmarks

go 1.25

require (
	github.com/fxamacker/cbor/v2 v2.9.0
	github.com/go-json-experiment/json v0.0.0-00010101000000-000000000000
	github.com/tinylib/msgp v1.5.0
	github.com/delaneyj/cbor v0.0.0
)

replace github.com/delaneyj/cbor => ../
replace github.com/go-json-experiment/json => ../json-experiment

