# Spec Delta

## MODIFIED Requirements

### Requirement: Stances and overlays
`jump`, `jump_forward`, `jump_back` and `duck` SHALL be stances. After a stance starts, the interpreter continues with the next statement on the following frame while the stance runs. While a stance is active:
- Sensor reads, `if`/`elif`/`else`, `repeat` bookkeeping and `wait` SHALL proceed concurrently.
- While airborne, the first `kick` SHALL become `jump_kick` and the first `punch` SHALL become `dive_punch`. Only one air attack is allowed per jump.
- While ducking (hold phase), `punch` SHALL become `low_jab`. More than one `low_jab` is allowed per duck.
- An overlay SHALL NOT reset or pause the stance's timer. An overlay attack occupies the interpreter until it finishes.
- Any other action, including a second air attack, SHALL wait until the stance has fully ended (including landing or duck recovery) and then execute normally.

An air attack still in progress when the fighter lands SHALL end on landing, unless it started in the landing-low zone (airborne frames 17–20). Such an attack SHALL continue to completion after touchdown, with landing recovery running at the same time, and the interpreter SHALL wait until both have finished. If a `low_jab` is in progress when the duck hold would expire, the hold SHALL extend until the `low_jab` finishes.

#### Scenario: Jump kick overlay
- **WHEN** a script is `jump_forward kick`
- **THEN** the jump starts on frame 0, `jump_kick` starts on frame 1 while airborne, and the jump still lands after 20 airborne frames

#### Scenario: One air attack per jump
- **WHEN** a script is `jump kick kick`
- **THEN** the first `kick` becomes `jump_kick`, and the second `kick` waits until landing recovery ends and then executes as a grounded `kick`

#### Scenario: Low jab overlay
- **WHEN** a script is `duck hold 20 punch`
- **THEN** the `punch` executes as `low_jab` once the duck reaches its hold phase, and the fighter stays ducking

#### Scenario: Non-overlay action waits for stance
- **WHEN** a script is `jump move_forward x2`
- **THEN** the movement begins only after the jump's 20 airborne frames and 2 landing frames have elapsed

#### Scenario: Checks run during a stance
- **WHEN** a script is `jump_forward if distance < 12 { kick }`
- **THEN** the jump starts on airborne frame 1, the `distance` read happens on airborne frames 2–3, and, if the read is true, `jump_kick` starts on airborne frame 4

#### Scenario: Falling air attack ends on landing
- **WHEN** a script is `jump wait 14 kick wait 1`, so the `jump_kick` starts on airborne frame 16
- **THEN** the jump kick ends at touchdown before finishing its active frames

#### Scenario: Landing-low attack carries through touchdown
- **WHEN** a script is `jump wait 16 kick wait 1`, so the `jump_kick` starts on airborne frame 18
- **THEN** the jump kick keeps all 5 of its active frames after touchdown, and the `wait` starts only after both the attack and landing recovery have finished
