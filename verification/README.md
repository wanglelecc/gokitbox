# Versioned acceptance suite

`*_test.go` remains ignored, as requested. The explicitly approved alternative is a
versioned test artifact. `foundation-suite-v1.tar.gz` packages the local acceptance
suite with its original package paths; it is not extracted into the working tree.
The JSON manifest binds its checksum to the exact production Go sources/go.mod/go.sum.

Run `python3 scripts/verify-foundation.py`. It checks source/archive fingerprints,
requires Go, CGO and redis-server, builds and runs all tests with race detection in a
temporary copy. Redis and Kafka tests use isolated servers/MockCluster, not production.
The GitHub workflow runs the same command. Artifact changes must be reviewed like test
source changes; the manifest is a consistency check, not an independent signature.

The suite contains compatibility, protocol, concurrent ownership, SIGKILL recovery,
resource-budget and secret-log assertions. Generation is explicit so an implementation
change cannot silently keep a stale green suite.
