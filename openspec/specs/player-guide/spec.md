# player-guide Specification

## Purpose

Defines `docs/PLAYER_GUIDE.md`, the document players and their LLMs use to write fight scripts. It must stay complete, correct and in sync with the engine, since it is the primary context an LLM receives through `get_player_guide`.

## Requirements

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
- interrupts and handler resume rules
- the script limits and the most common validation errors with their fixes
- strategy notes drawn from the sample bots
- how to play: through MCP (`pf mcp` tools) and through the CLI and HTTP

It SHALL state the config version it describes.

#### Scenario: Guide states its version
- **WHEN** the guide is read
- **THEN** it names config version `2026.11-2`

### Requirement: Guide stays in sync with the engine
An automated test SHALL fail when any of these are missing from the guide:
- every action keyword
- every sensor name
- every attack name accepted by `in_range`
- each attack's startup, active and recovery values from the default config
- each script limit (bytes, statements, depth)

#### Scenario: New sensor without docs
- **WHEN** a sensor is added to the language vocabulary but not to the guide
- **THEN** the guide sync test fails and names the missing sensor

### Requirement: Guide examples are valid
Every fenced code block in the guide tagged `pf` SHALL be a valid script under the default limits, which an automated test enforces. Examples of mistakes SHALL use a different tag.

#### Scenario: Broken example
- **WHEN** a `pf` example in the guide contains `kik`
- **THEN** the guide examples test fails and reports the block and the validation error
