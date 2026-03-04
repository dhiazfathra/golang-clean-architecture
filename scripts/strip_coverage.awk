# strip_coverage.awk — mirrors codecov.yml `ignore:` patterns
# Usage: awk -f scripts/strip_coverage.awk coverage.out > coverage.out.tmp
#
# Each rule drops lines whose file path matches a codecov.yml ignore entry:
#
#   **/*_test.go          → Go never instruments test files; not in coverage.out
#   **/mock_*.go          → /mock_[filename].go
#   **/mocks/**           → /mocks/ directory
#   **/testdata/**        → Go skips testdata/ by convention; not in coverage.out
#   **/*.pb.go            → .pb.go suffix
#   **/*.gen.go           → .gen.go suffix
#   **/vendor/**          → go list excludes vendor; not in coverage.out
#   cmd/*/main.go         → /cmd/[name]/main.go

/^mode:/                   { print; next }
/\/mocks\//                { next }
/\/testdata\//             { next }
/\.pb\.go:/                { next }
/\.gen\.go:/               { next }
/\/vendor\//               { next }
/\/mock_[^\/]*\.go:/       { next }
/\/[^\/]*_mock\.go:/       { next }
/\/cmd\/[^\/]+\/main\.go:/ { next }
                           { print }
