# Spec Delta

## ADDED Requirements

### Requirement: pf serve
`pf serve [--addr HOST:PORT] [--data DIR]` SHALL start the HTTP API server. `--addr` defaults to `127.0.0.1:8080` and `--data` defaults to `./data`, which SHALL be created if missing. It SHALL log one line on startup naming the listen address and data directory. On SIGINT or SIGTERM it SHALL shut down gracefully, letting in-flight requests finish, and exit 0. An unusable address or data directory SHALL print an error to stderr and exit 2.

#### Scenario: Default bind is local
- **WHEN** `pf serve` is run with no flags
- **THEN** it listens on `127.0.0.1:8080` and stores data under `./data`

#### Scenario: Graceful shutdown
- **WHEN** the server receives SIGINT while idle
- **THEN** it exits 0

#### Scenario: Bad address
- **WHEN** `pf serve --addr nonsense` is run
- **THEN** it prints an error to stderr and exits 2
