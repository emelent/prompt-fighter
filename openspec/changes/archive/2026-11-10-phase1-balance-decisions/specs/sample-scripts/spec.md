# Spec Delta

## MODIFIED Requirements

### Requirement: Required sample scripts
The repository SHALL include at least these valid scripts in `/scripts`, each with header comments describing its strategy:
- `turtle.pf`: spends most of its time blocking and punishes after blocks.
- `rushdown.pf`: closes distance quickly and attacks constantly.
- `jumper.pf`: relies primarily on jumps and air attacks.
- `thrower.pf`: closes distance and uses throws as its main damage.
- `lazy.pf`: contains no sensor reads at all.
- `ducker.pf`: relies on `duck` to avoid grounded high attacks and counters with low attacks (`low_jab`, `sweep`).

#### Scenario: All samples validate
- **WHEN** `pf validate scripts/*.pf` is run
- **THEN** every file reports `ok`

#### Scenario: Lazy has no checks
- **WHEN** `lazy.pf` is simulated against any opponent
- **THEN** the log contains no `check` events with actor equal to lazy's player id

#### Scenario: Ducker ducks
- **WHEN** `ducker.pf` is simulated against any other sample
- **THEN** its fighter is in the `duck` state on at least one frame
