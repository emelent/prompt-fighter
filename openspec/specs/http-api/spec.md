# http-api Specification

## Purpose

Defines the HTTP/JSON API served by `pf serve`: health, script validation and single-round simulation endpoints, the shared error envelope, and the request limits that keep a trusted local server responsive.

## Requirements

### Requirement: JSON transport and error envelope
Every API endpoint (`/healthz`, `/validate`, `/sim` and everything under `/matches`) SHALL accept and return `application/json`. The only non-JSON responses are the web app's pages and assets: the replay viewer under `/view` (see the replay-viewer capability), the play pages under `/play` (see the play-ui capability), and the `GET /` redirect to `/play`. Every error response, including a 404 for an unknown route or viewer asset, SHALL have the body `{"error": {"code": <code>, "message": <text>}}`, with one of these codes and statuses:

| code | status |
|---|---|
| `bad_request` | 400 (malformed JSON, missing or ill-typed fields, values out of range) |
| `unauthorized` | 401 |
| `not_found` | 404 (unknown route or resource) |
| `method_not_allowed` | 405 |
| `conflict` | 409 |
| `payload_too_large` | 413 |
| `invalid_script` | 422 |
| `busy` | 503 |
| `internal` | 500 (an unexpected server-side failure, such as a storage write error; never returned for client mistakes) |

An `invalid_script` error SHALL also carry `errors`: a list of `{"player"?, "line", "col", "message", "text"}`. `text` is the `line:col: message` rendering used by `pf validate`, and `player` names the side or seat when a request carries more than one script. Unknown JSON fields SHALL be rejected with `bad_request`.

#### Scenario: Malformed JSON
- **WHEN** a client sends `POST /validate` with the body `{"script": `
- **THEN** the response is 400 with `error.code` `bad_request`

#### Scenario: Unknown route
- **WHEN** a client sends `GET /nope`
- **THEN** the response is 404 with `error.code` `not_found` in the JSON envelope

#### Scenario: Wrong method
- **WHEN** a client sends `GET /sim`
- **THEN** the response is 405 with `error.code` `method_not_allowed`

#### Scenario: Unknown field
- **WHEN** a client sends `POST /validate` with `{"script": "kick", "scirpt": "x"}`
- **THEN** the response is 400 with `error.code` `bad_request` and a message naming the unknown field

#### Scenario: Viewer page is HTML
- **WHEN** a client sends `GET /view`
- **THEN** the response is 200 with content type `text/html; charset=utf-8`, not JSON

#### Scenario: Play page is HTML
- **WHEN** a client sends `GET /play`
- **THEN** the response is 200 with content type `text/html; charset=utf-8`, not JSON

#### Scenario: Root redirects to the lobby
- **WHEN** a client sends `GET /`
- **THEN** the response is a 302 redirect to `/play`, while other unknown paths such as `GET /nope` still return the JSON 404

### Requirement: Health endpoint
`GET /healthz` SHALL return 200 with `{"status": "ok", "config_version": <engine default config version>, "format_version": <event log format version>, "store_version": <match store version>}`.

#### Scenario: Health reports versions
- **WHEN** a client sends `GET /healthz` to a server built at config `2026.11-2`
- **THEN** the response is 200 with `status` `ok`, `config_version` `2026.11-2` and `format_version` `1`

### Requirement: Validate endpoint
`POST /validate` with `{"script": <source>}` SHALL validate the script under the engine's default limits and return 200 with `{"valid": <bool>, "errors": [...]}`. Each error is `{"line", "col", "message", "text"}`, in the same order and with the same text that `pf validate` prints. An invalid script is a successful validation, not an HTTP error.

#### Scenario: Valid script
- **WHEN** a client posts `{"script": "loop { kick }"}` to `/validate`
- **THEN** the response is 200 with `valid` true and an empty `errors` list

#### Scenario: Invalid script
- **WHEN** a client posts a script whose line 3, column 5 is `kik` to `/validate`
- **THEN** the response is 200 with `valid` false and one error with `line` 3, `col` 5 and `text` `3:5: unknown action "kik" — did you mean "kick"?`

### Requirement: Sim endpoint
`POST /sim` with `{"p1": {"name"?, "script"}, "p2": {"name"?, "script"}, "frames"?}` SHALL run one round under the default config, with p1 and p2 as given, and return 200 with exactly the bytes of the engine's compact event log JSON for those inputs:
- `name` is optional, defaults to `p1`/`p2`, and is at most 40 bytes.
- `frames` is optional, defaults to the server's frame budget, and MUST be from 1 to that budget.
- If either script is invalid, the response SHALL be 422 `invalid_script`, with each error tagged `player` `p1` or `p2`.

#### Scenario: Log equals the engine output
- **WHEN** a client posts two valid scripts to `/sim`
- **THEN** the response body is byte-identical to the engine's log for the same names, scripts and frame budget

#### Scenario: Custom frames
- **WHEN** a client posts two valid scripts with `"frames": 120` and no KO occurs
- **THEN** the returned log has `frame_budget` 120 and 120 frames

#### Scenario: Frames out of range
- **WHEN** a client posts `"frames": 0` or `"frames": 601`
- **THEN** the response is 400 `bad_request`

#### Scenario: Invalid script in sim
- **WHEN** a client posts a valid p1 script and a p2 script containing `kik`
- **THEN** the response is 422 `invalid_script` with an error whose `player` is `p2`

### Requirement: Request limits
The server SHALL reject request bodies larger than 64 KiB with 413 `payload_too_large`. It SHALL run at most a configured number of simulations at once (default: the number of CPUs). A request that cannot start its simulation within 5 seconds SHALL receive 503 `busy`. The server SHALL set read-header, read and write timeouts so that a stalled client cannot hold a connection indefinitely.

#### Scenario: Oversized body
- **WHEN** a client posts a 65 KiB body to `/sim`
- **THEN** the response is 413 `payload_too_large` and no simulation runs

#### Scenario: Saturated simulator
- **WHEN** every simulation slot stays occupied for more than 5 seconds while another `/sim` request waits
- **THEN** the waiting request receives 503 `busy`
