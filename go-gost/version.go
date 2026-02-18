package main

var (
	version = "0.3.1"
)

func normalizedVersion() string {
	v := version
	if v == "" {
		return "0.3.1"
	}
	return v
}
