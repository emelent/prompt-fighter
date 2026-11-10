# Prompt Fighter — Engine Rules (config `2026.11-1`, log format `1`)

This is the reference for how the Phase 1 simulation engine behaves. It records every rule, including each decision taken where the build spec was silent; those are marked **[gap-fill]**. All tunables live in `sim.DefaultConfig()`. Changing any default bumps the config version.

## 1. Stage and round

| item | value |
|---|---|
| stage | x = 0..100 units |
| start positions | p1 x=30, p2 x=70, y=0 |
| fighter half-width | 2. Centres stay in [2, 98] and are never closer than 4 **[gap-fill]** |
| HP | 100 each |
| frame budget | 600 frames (`pf sim --frames N` overrides it) |
| arithmetic | integers only. No floats, randomness or map-order dependence |

A round ends at the end of the first frame on which any fighter's HP reaches 0 (**KO**), or when the frame budget runs out (**timeout**).
- KO: the fighter with HP left wins. If both reach 0 on the same frame, the result is a draw by `double_ko`. No frames are simulated after the KO frame.
- Timeout: higher HP wins; equal HP is a draw.
- HP never goes below 0.

Fighters never pass through or over each other. Airborne fighters still collide, so cross-ups are impossible and facing never changes in practice **[gap-fill]**. Grounded fighters always face each other, and facing is frozen while airborne.

## 2. Frame data

Phases run in the order startup → active → recovery. Range is centre-to-centre distance.

| attack | startup | active | recovery | total | height | damage | range | knockdown |
|---|---|---|---|---|---|---|---|---|
| punch | 3 | 2 | 5 | 10 | high | 5 | 8 | |
| kick | 6 | 3 | 9 | 18 | high | 9 | 12 | |
| sweep | 8 | 3 | 14 | 25 | low | 8 | 12 | yes |
| throw | 5 | 1 | 15 | 21 | — (unblockable) | 12 | 5 | yes |
| jump_kick (overlay) | 3 | 5 | 0 | 8 | high | 10 | 10 | |
| dive_punch (overlay) | 2 | 4 | 0 | 6 | high | 6 | 8 | |
| low_jab (overlay) | 3 | 2 | 5 | 10 | low | 4 | 7 | |

The jump_kick, dive_punch and low_jab rows are **[gap-fill]** values. jump_kick has to fit inside a 20-frame jump. dive_punch is faster but weaker. low_jab mirrors punch.

| other | value |
|---|---|
| walk (`move_forward`/`move_back`) | 1 unit per frame, 1 frame per unit of `xN` (default 1) |
| jump | 20 airborne frames (10 rising, 10 falling) + 2 landing frames. `jump_forward`/`jump_back` move 1 unit per airborne frame. y rises 2 per frame to a peak of 20 **[gap-fill: y velocity is visual only]** |
| duck | 2 startup + hold + 2 recovery. Only the hold phase counts as ducking |
| high_block / low_block | 1 startup + hold + 1 recovery. Only the hold phase guards |
| default hold (when `hold N` is omitted) | 10 frames **[gap-fill]** |
| hitstun | 5 frames after the hit frame, with no script input |
| hit pushback | 3 units |
| defender blockstun | 3 frames. The defender keeps guarding, and its hold is extended to cover the blockstun |
| block pushback | 2 units |
| knockdown | 20 frames after the knockdown frame. Invulnerable, no script input |
| sensor read | 2 frames |
| counts (`xN`, `hold N`, `wait N`, `repeat N`) | 1–100 **[gap-fill]**. Condition literals are 0–999 |

## 3. Tick order

Every frame both fighters are processed simultaneously and symmetrically. Each phase computes results for both fighters from the same input before anything changes, so swapping sides mirrors the round exactly.

1. **Snapshot.** Each fighter's sensor-visible state at the end of the previous frame is captured. All sensor reads this frame see this snapshot.
2. **Stance bookkeeping.** Jumps and ducks that have finished end. A `low_jab` in progress extends the duck hold. Air attacks still in progress end on landing **[gap-fill]**.
3. **Think.** Each fighter that is not stunned and not busy advances its script until something occupies it for the frame: an action, a sensor read, or an idle frame.
4. **Move.** Both fighters compute their new x (walking and jump drift), clamped to the stage. If the new positions would be closer than 4, every fighter that moved toward the other this frame has its move cancelled.
5. **Connect.** Each active attack that has not yet connected is checked against the post-move positions (§5).
6. **Resolve.** All connections apply together (§6).
7. **Whiffs.** An attack whose last active frame ends without connecting emits `whiff`.
8. **KO check.** A `ko` event is emitted for each fighter at 0 HP.
9. **Record.** The frame is recorded, then finished actions retire and the script moves to its next statement.

Events are ordered by frame, then p1's events before p2's, then in emission order. That ordering is the engine's only side asymmetry, and it never affects game state.

## 4. Scripts over time

- An attack occupies the fighter for its total frames. A move costs N frames, `wait N` costs N frames, and a block costs 1 + hold + 1 frames.
- A stance start (`jump`, `jump_forward`, `jump_back`, `duck`) costs 1 frame. The next statement begins on the following frame while the stance continues in the background **[gap-fill]**.
- `if`, `elif`, `else`, `repeat` and `loop` bookkeeping is free. Each **sensor read costs 2 frames**. Literals are free. `and`/`or` short-circuit left to right, and sensors that are never evaluated cost nothing.
- A sensor's value is taken from the snapshot on the first frame of its read. A change on the second frame is not seen.
- **Zero-cost loop guard.** If a `loop` iteration finishes without any frame passing, its back-edge costs 1 idle frame. A script can never spin without time passing.
- When the main program finishes, `script_end` is emitted once and the fighter idles for the rest of the round.

### Stances and overlays
While a stance is active:
- Sensor reads, `wait` and control flow carry on as normal.
- While airborne, the first `kick` becomes `jump_kick` and the first `punch` becomes `dive_punch`. Only one air attack is allowed per jump.
- While in the duck hold, `punch` becomes `low_jab`, and any number of jabs is allowed.
- An overlay never pauses the stance's timer.
- Any other action, including a second air attack, waits until the stance has fully ended (including landing or duck recovery) and then runs normally. Those waiting frames show the stance animation.

### Interrupts and handlers
- **Hit** (hitstun, or knockdown including from a throw). The current action and stance are abandoned. When the fighter can act again, `on_hit` runs if it is defined, and then the script resumes at the statement after the interrupted one. Waking up from a knockdown counts as leaving the interrupt, so `on_hit` runs then too **[gap-fill]**.
- **Your attack was blocked** **[gap-fill, see §6]**. The attack is abandoned and you enter blockstun. When the blockstun ends, `on_block` runs, then the script resumes after the attack.
- **"The statement after"** is worked out structurally. Inside a block it is the next statement. If the interrupt came during an `if`/`elif` condition, the whole `if` chain is skipped. If the interrupted statement was the last one in a `repeat` or `loop`, the next iteration starts (or the `repeat` ends).
- **Interrupted inside a handler.** The handler is abandoned, then restarts from the top after the new stun. The resume point stays the one saved by the first interrupt.
- `handler_enter` is emitted every time a handler starts. Handlers still run after `script_end`, and afterwards the fighter goes back to idling.

### Sensors

| sensor | type | meaning (from the reader's side) |
|---|---|---|
| `distance` | int | \|x₁ − x₂\| |
| `health`, `enemy.health` | int | current HP |
| `in_range(a)` | bool | `distance` ≤ range of attack `a` (any of the 7 attacks above) |
| `enemy.airborne` | bool | the enemy is in jump frames 1–20 (not landing) |
| `enemy.ducking` | bool | the enemy is in the duck hold phase, including during a `low_jab` |
| `enemy.blocking` | bool | the enemy is in a block's hold phase or in blockstun. Block startup and recovery don't count **[gap-fill]** |
| `enemy.attacking` | bool | the enemy has an attack (overlays included) in startup or active. Recovery doesn't count **[gap-fill]** |
| `enemy.knocked_down` | bool | the enemy is in knockdown |

**Reaction windows.** Because reads cost 2 frames and see the previous frame's state, a script that polls `loop { if enemy.attacking { high_block } }` blocks every kick and every sweep (using `low_block` for the sweep), whatever frame the attack starts on. It can never react to a punch: 3 startup frames is shorter than the read plus the block startup. The engine tests confirm this.

## 5. Connection and whiffs

On each active frame, an attack that has not yet connected connects if the target is within range and none of these apply. When one does apply, it becomes the whiff `reason`:

| reason | condition |
|---|---|
| `out_of_range` | distance > range |
| `invulnerable` | target is knocked down |
| `ducked` | high attack against a target in the duck hold |
| `airborne` | low attack or throw against an airborne target |
| `stunned` | throw against a target in hitstun |

An attack connects at most once. If its active frames end without connecting, it emits `whiff` with the reason from its last active frame.

## 6. Resolution

All connections on a frame are resolved together from the pre-resolution state.

- **Attack beats throw.** If a fighter is hit before or on its throw's active frame, the throw is abandoned. If an attack and a throw would both connect on the same frame, only the attack applies.
- **Throw tech.** If both throws connect on the same frame, there is no damage and both fighters are pushed back 2. Each emits `throw` with outcome `tech`, and both throws continue into recovery **[gap-fill]**.
- **Hit.** The target takes damage, enters hitstun (or knockdown, if the attack knocks down) and is pushed back 3.
- **Anti-air.** Any attack that hits an airborne target knocks it down and snaps its y to 0. There are no air juggles **[gap-fill]**.
- **Block.** An attack that meets a target guarding its height during the hold phase is blocked.
  - Defender: no damage, 3 frames of blockstun with the guard held, pushed back 2. The defender is not interrupted.
  - Attacker: the attack is abandoned, and the attacker is stunned for the attack's **recovery** length. It then runs `on_block` **[gap-fill]**.

  This reading of the build spec ("`on_block` runs after blockstun when *your* attack was blocked") keeps blocks beating attacks. A flat 3-frame stun would make a blocked attack recover faster than a whiff. A block of the wrong height does not guard. Throws ignore blocks.
- **Throw success.** 12 damage and a knockdown **[gap-fill]**. The thrower recovers on the frame after its 15 recovery frames, 5 frames before the target wakes up. An immediately repeated throw therefore becomes active exactly on the wake-up frame. **Balance note:** at the defaults, `loop { throw }` throw-loops an idle opponent, and also one running `loop { jump }`, to a KO. Attacking on wake-up escapes it, because attacks beat throws. This is flagged for the Phase 1 review. Design D8 assumed the opposite.
- **Trade.** If both attacks hit on the same frame, both apply, and a `trade` event is emitted on top of the two `hit` events.
- **Pushback at a wall.** Any pushback a wall prevents is transferred to the other fighter, in the opposite direction **[gap-fill]**.

### Combos
A hit landing while the defender is still in hitstun from an earlier hit increases the defender's combo count. The count resets when the defender can act again or is knocked down.
- Damage multiplier: 100% for the first hit, then `previous × 80 / 100` (integer division), never below 30%. That gives 100, 80, 64, 51, 40, 32, 30…
- Damage is `max(1, base × mult / 100)`.
- The 5th hit of a combo causes a knockdown.

At the default values, combos are effectively dormant. Hitstun (5 frames) is shorter than any attacker's recovery plus the next startup, and pushback adds distance. This is flagged for the Phase 1 balance review.

### Knockdown
20 frames on the ground, invulnerable to attacks and throws, with no script input. The fighter then stands up idle, can be hit again and can act (and runs `on_hit`).

## 7. Event log

The log is a single JSON object with these fields:
- `format_version`, `config_version`, `frame_budget`
- `players`: `[{id, name, script}]` × 2
- `result`: `{winner: "p1"|"p2"|null, reason: ko|double_ko|timeout, end_frame, hp: [p1, p2]}`
- `frames`
- `events`

`frames[i]` is the state at the end of frame i, for i = 0..end_frame. Each entry holds `f` plus a `p1` and a `p2` object with these fields:

| field | values |
|---|---|
| `x`, `y` | integers |
| `facing` | 1 (facing +x) or −1 |
| `state` | idle, walk, attack, block, duck, airborne, landing, hitstun, blockstun, knockdown, ko |
| `anim` | idle, check, wait, walk_forward, walk_back, punch, kick, sweep, throw, jump, jump_kick, dive_punch, duck, low_jab, high_block, low_block, landing, hitstun, blockstun, knockdown, ko |
| `anim_frame` | frames since `anim` (for this action instance) began, from 0 |
| `phase` | startup, active, recovery, or "" |
| `hp`, `combo` | integers. `combo` is this fighter's current combo count as defender |
| `invulnerable` | true during knockdown |
| `script_line` | 1-based line of the statement being executed, or 0 when stunned, KO'd or after the script ended |

During a sensor read, `anim` is `check` unless a stance is active, in which case the stance animation is shown. A defender in blockstun shows `state: blockstun` with its block animation.

### Events
Each event is `{frame, type, actor, target?, detail}`. `detail` fields by type:

| type | actor | target | detail |
|---|---|---|---|
| `hit` | attacker | defender | `attack`, `damage`, `combo`, `hp` (defender's HP after) |
| `block` | attacker | defender | `attack` |
| `whiff` | attacker | — | `attack`, `reason` (§5) |
| `throw` | thrower | defender (success only) | `attack: "throw"`, `outcome: success|tech`, plus `damage` and `hp` on success |
| `trade` | p1 | p2 | `attacks: [p1's, p2's]`, `damages: [to p2, to p1]` |
| `knockdown` | fighter knocked down | — | `attack` (what caused it), `reason`: `airborne` / `combo` / `sweep` / `throw` |
| `ko` | KO'd fighter | — | (empty) |
| `check` | reader | — | `sensor` (e.g. `distance`, `in_range(kick)`), `value` (int or bool, always present), `line`, `cost` |
| `script_end` | fighter | — | (empty) |
| `handler_enter` | fighter | — | `handler`: `on_hit` / `on_block` |

`damage`, `hp` and `value` are always written when they apply, even when they are 0 or false. Other zero-valued fields are omitted.

Determinism: the same scripts, config and frame budget always produce byte-identical JSON. Swapping p1 and p2 produces the mirrored round (x → 100 − x, facing negated, players exchanged, same winning script).

## 8. Script limits
- At most 4096 bytes.
- At most 40 statements. Each action, `if`, `elif`, `repeat` and `loop` counts as 1, across the main program and handlers. `else` and handler declarations count 0 **[gap-fill]**.
- Nesting depth of at most 4, where a handler block counts as one level.
- At most one `on_hit` and one `on_block`, both at top level. There is no `loop` inside a handler, and a `loop` body can't be empty.
- Validation reports up to 10 errors, sorted, as `line:col: message`.
