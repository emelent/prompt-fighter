# Spec Delta

## Purpose

Defines the bundled sample fight scripts. They demonstrate the language, act as test fixtures and later serve as dummy opponents, and together they form a non-transitive set useful as a balance baseline.

## ADDED Requirements

### Requirement: Required sample scripts
The repository SHALL include at least these valid scripts in `/scripts`, each with header comments describing its strategy:
- `turtle.pf`: spends most of its time blocking and punishes after blocks.
- `rushdown.pf`: closes distance quickly and attacks constantly.
- `jumper.pf`: relies primarily on jumps and air attacks.
- `thrower.pf`: closes distance and uses throws as its main damage.
- `lazy.pf`: contains no sensor reads at all.

#### Scenario: All samples validate
- **WHEN** `pf validate scripts/*.pf` is run
- **THEN** every file reports `ok`

#### Scenario: Lazy has no checks
- **WHEN** `lazy.pf` is simulated against any opponent
- **THEN** the log contains no `check` events with actor equal to lazy's player id

### Requirement: Non-transitive balance among samples
Under the default config, each sample script SHALL beat at least one other sample and lose to at least one other sample. A script "beats" another when it has more wins than losses against it across both side assignments in `pf selfplay`. This SHALL be enforced by an automated test.

#### Scenario: Every sample has a win and a loss
- **WHEN** the selfplay round-robin is computed over `/scripts` in a test
- **THEN** every script has at least one opponent it beats and at least one opponent it loses to
