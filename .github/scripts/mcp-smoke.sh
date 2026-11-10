#!/usr/bin/env bash
# Smoke test for `pf mcp` over stdio, as an MCP client sees it:
#   1. the handshake and the tool list, with no server;
#   2. a local tool call (validate_script);
#   3. a real match against a running `pf serve`: create_match in one session,
#      join_match in another (which plays round 1), then get_round_replay.
# Usage: mcp-smoke.sh <path-to-pf> <server-url>
set -euo pipefail

PF=$1
SERVER=$2
SCRIPT=$(cat "$(dirname "$0")/../../scripts/turtle.pf")

# rpc <server-url> <request-json>...: runs one MCP session and prints its responses.
rpc() {
  local server=$1
  shift
  {
    printf '%s\n' '{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2026-07-28","capabilities":{},"clientInfo":{"name":"ci-smoke","version":"1"}}}'
    printf '%s\n' '{"jsonrpc":"2.0","method":"notifications/initialized"}'
    printf '%s\n' "$@"
    sleep 2 # keep stdin open until the server has answered
  } | "$PF" mcp --server "$server" 2>/dev/null
}

call() { # call <id> <tool> <arguments-json>
  jq -cn --argjson id "$1" --arg name "$2" --argjson args "$3" '{jsonrpc:"2.0",id:$id,method:"tools/call",params:{name:$name,arguments:$args}}'
}

result() { # result <id> < responses: the result of the response with that id
  jq -ce --argjson id "$1" 'select(.id == $id) | .result'
}

fail() { echo "FAIL: $*" >&2; exit 1; }

echo "== handshake and tools"
out=$(rpc "" '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' "$(call 2 validate_script "$(jq -cn --arg s "$SCRIPT" '{script:$s}')")")
echo "$out" | result 0 | jq -e '.protocolVersion and .capabilities.tools' >/dev/null || fail "initialize"
tools=$(echo "$out" | result 1 | jq -r '[.tools[].name] | sort | join(",")')
want="create_match,get_match,get_player_guide,get_round_replay,join_match,list_sample_bots,simulate_round,submit_script,validate_script"
[ "$tools" = "$want" ] || fail "tools: got $tools, want $want"
echo "$out" | result 2 | jq -e '.isError != true and .structuredContent.valid == true' >/dev/null || fail "validate_script"
echo "ok: $tools"

echo "== match against $SERVER"
args=$(jq -cn --arg s "$SCRIPT" '{name:"ci-a",script:$s}')
id=$(rpc "$SERVER" "$(call 1 create_match "$args")" | result 1 | jq -er '.structuredContent.match_id') || fail "create_match"
args=$(jq -cn --arg id "$id" --arg s "$SCRIPT" '{match_id:$id,name:"ci-b",script:$s}')
rpc "$SERVER" "$(call 1 join_match "$args")" | result 1 | jq -e '.isError != true and .structuredContent.match_id' >/dev/null || fail "join_match"
rpc "$SERVER" "$(call 1 get_round_replay "$(jq -cn --arg id "$id" '{match_id:$id,round:1}')")" | result 1 | jq -e '.isError != true' >/dev/null || fail "get_round_replay"
echo "ok: match $id played round 1"
