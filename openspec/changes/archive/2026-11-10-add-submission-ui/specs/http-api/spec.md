# Spec Delta

## MODIFIED Requirements

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
