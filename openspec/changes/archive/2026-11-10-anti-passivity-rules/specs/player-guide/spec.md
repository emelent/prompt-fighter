# Spec Delta

## MODIFIED Requirements

### Requirement: Guide content
`docs/PLAYER_GUIDE.md` SHALL teach a script author, without requiring `RULES.md`:
- the goal of a round and how it ends
- the script structure (main program, `on_hit` and `on_block`)
- every action, with its argument form and frame cost
- every sensor, with its meaning and 2-frame cost
- the condition syntax
- `if`/`elif`/`else`, `repeat` and `loop`
- stances and overlays, including the air-attack zones and the landing-low carry-through
- the frame data for every attack
- hits, blocks and blockstun, the throw triangle including on wake-up, and combos
- guard breaks, the passivity boost and stale attacks, with how to avoid them
- interrupts and handler resume rules
- the script limits and the most common validation errors with their fixes
- strategy notes drawn from the sample bots
- how to play: through MCP (`pf mcp` tools) and through the CLI and HTTP

It SHALL state the config version it describes.

#### Scenario: Guide states its version
- **WHEN** the guide is read
- **THEN** it names config version `2026.11-3`
