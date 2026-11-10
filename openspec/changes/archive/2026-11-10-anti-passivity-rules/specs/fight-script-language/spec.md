# Spec Delta

## MODIFIED Requirements

### Requirement: Conditions
A condition SHALL be an expression built from:
- Boolean sensors: `enemy.airborne`, `enemy.ducking`, `enemy.blocking`, `enemy.attacking`, `enemy.knocked_down`, and `in_range(<attack>)`, where `<attack>` is one of `punch`, `kick`, `sweep`, `throw`, `jump_kick`, `dive_punch`, `low_jab`.
- Integer sensors: `distance`, `health`, `enemy.health`, `guard`, `passive`, `enemy.passive`, `repeats`.
- Integer literals from 0 to 999.
- Comparisons `<`, `<=`, `>`, `>=`, `==`, `!=` between two integer operands.
- `not`, `and`, `or`, and parentheses.

Precedence from highest to lowest SHALL be: parentheses, comparison, `not`, `and`, `or`. Type errors SHALL be rejected: a boolean used in a comparison, an integer used as a boolean, or a comparison chained onto another comparison.

#### Scenario: Valid compound condition
- **WHEN** a condition is `(distance <= 12 or enemy.attacking) and not enemy.knocked_down`
- **THEN** validation succeeds

#### Scenario: Integer used as boolean
- **WHEN** a condition is `if health { kick }`
- **THEN** validation fails with a message stating that `health` is a number and must be compared, e.g. `health < 30`

#### Scenario: Boolean compared to number
- **WHEN** a condition is `enemy.airborne == 1`
- **THEN** validation fails with a message stating that `enemy.airborne` is true/false and should be used directly, e.g. `if enemy.airborne { ... }`

#### Scenario: Unknown sensor with suggestion
- **WHEN** a condition is `enemy.jumping`
- **THEN** validation fails at that token with a message listing it as an unknown sensor and suggesting `enemy.airborne`

#### Scenario: in_range with a non-attack argument
- **WHEN** a condition is `in_range(jump)`
- **THEN** validation fails with a message listing the valid attack names for `in_range`

#### Scenario: Anti-passivity sensors
- **WHEN** a condition is `passive > 90 or guard >= 3 or repeats >= 5`
- **THEN** validation succeeds
