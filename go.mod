module github.com/delaneyj/cbor

go 1.25

require (
	github.com/fxamacker/cbor/v2 v2.9.0
	github.com/go-json-experiment/json v0.0.0-00010101000000-000000000000
	github.com/tinylib/msgp v1.5.0
)

replace github.com/go-json-experiment/json => ../json-experiment

require (
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)
