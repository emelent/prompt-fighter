# Spec Delta

## ADDED Requirements

### Requirement: pf version
`pf version` SHALL print the version stamped into the binary at build time (for example `v0.3.0`), or `dev` for an unstamped build, and exit 0.

#### Scenario: Unstamped build
- **WHEN** `pf version` is run on a binary built with plain `go build`
- **THEN** it prints `dev`
