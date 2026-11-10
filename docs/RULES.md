# Prompt Fighter — Engine Rules (config `2026.11-3`, log format `1`)

This is the reference for how the Phase 1 simulation engine behaves. It records every rule, including each decision taken where the build spec was silent; those are marked **[gap-fill]**. All tunables live in `sim.DefaultConfig()`. Changing any default bumps the config version. Open balance questions are tracked in [`ISSUES.md`](ISSUES.md).

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
| throw | 5 | 1 | 18 | 24 | — (unblockable) | 12 | 5 | yes |
| jump_kick (overlay) | 3 | 5 | 0 | 8 | by zone (§5) | 10 | 10 | |
| dive_punch (overlay) | 2 | 4 | 0 | 6 | by zone (§5) | 6 | 8 | |
| low_jab (overlay) | 3 | 2 | 5 | 10 | low | 4 | 7 | |

The jump_kick, dive_punch and low_jab rows are **[gap-fill]** values. jump_kick has to fit inside a 20-frame jump. dive_punch is faster but weaker. low_jab mirrors punch.

| other | value |
|---|---|
| walk (`move_forward`/`move_back`) | 1 unit per frame, 1 frame per unit of `xN` (default 1) |
| jump | 20 airborne frames (10 rising, 10 falling) + 2 landing frames. `jump_forward`/`jump_back` move 1 unit per airborne frame. y rises 2 per frame to a peak of 20 **[gap-fill: y velocity is visual only]** |
| duck | 2 startup + hold + 2 recovery. Only the hold phase counts as ducking |
| high_block / low_block | 1 startup + hold + 1 recovery. Only the hold phase guards |
| default hold (when `hold N` is omitted) | 10 frames **[gap-fill]** |
| hitstun | 13 frames after the hit frame, with no script input (long enough for short combos, §6) |
| hit pushback | 2 units |
| defender blockstun | 3 frames. The defender keeps guarding, and its hold is extended to cover the blockstun |
| block pushback | 2 units |
| knockdown | 20 frames after the knockdown frame. Invulnerable, no script input |
| guard break | the 4th attack guarded in a row (blocks less than 60 frames apart) breaks the guard: 20 frames of stun, no script input, can't guard (§6) |
| passivity | 120 frames without starting an attack makes the opponent's next hit or throw deal ×1.1, rising 0.1 every 30 frames to ×1.5 (§6) |
| stale attacks | from the 6th start of the same attack in a row, its range is 85%, 70%, 55%, then 40% (§5) |
| sensor read | 2 frames |
| counts (`xN`, `hold N`, `wait N`, `repeat N`) | 1–100 **[gap-fill]**. Condition literals are 0–999 |

## 3. Tick order

Every frame both fighters are processed simultaneously and symmetrically. Each phase computes results for both fighters from the same input before anything changes, so swapping sides mirrors the round exactly.

1. **Snapshot.** Each fighter's sensor-visible state at the end of the previous frame is captured. All sensor reads this frame see this snapshot.
2. **Stance bookkeeping.** Jumps and ducks that have finished end. A `low_jab` in progress extends the duck hold. Air attacks still in progress end on landing **[gap-fill]**, except landing-low attacks (§5), which carry through touchdown and finish on the ground.
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
- While airborne, the first `kick` becomes `jump_kick` and the first `punch` becomes `dive_punch`. Only one air attack is allowed per jump. What the air attack can hit depends on the airborne frame it starts on (§5, air-attack zones).
- While in the duck hold, `punch` becomes `low_jab`, and any number of jabs is allowed.
- An overlay never pauses the stance's timer.
- Any other action, including a second air attack, waits until the stance has fully ended (including landing or duck recovery) and then runs normally. Those waiting frames show the stance animation.

### Interrupts and handlers
- **Hit** (hitstun, or knockdown including from a throw), and **guard break** (§6). The current action and stance are abandoned. When the fighter can act again, `on_hit` runs if it is defined, and then the script resumes at the statement after the interrupted one. Waking up from a knockdown counts as leaving the interrupt, so `on_hit` runs then too **[gap-fill]**.
- **Your attack was blocked** **[gap-fill, confirmed in the Phase 1 review; see §6]**. The attack is abandoned and you enter blockstun. When the blockstun ends, `on_block` runs, then the script resumes after the attack.
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
| `guard` | int | your block streak: attacks guarded in a row, 0–3. The next guarded attack at 3 breaks your guard (§6) |
| `passive`, `enemy.passive` | int | frames of passivity (yours, the enemy's), capped at 999. At 120 or more, the other fighter's next hit or throw is boosted (§6) |
| `repeats` | int | how many times in a row you have started your most recent attack (0 before your first). At 5 or more, starting it again is stale (§5) |

**Reaction windows (intended).** Because reads cost 2 frames and see the previous frame's state, a script that polls `loop { if enemy.attacking { high_block } }` blocks every kick and every sweep (using `low_block` for the sweep), whatever frame the attack starts on. It can never react to a punch: 3 startup frames is shorter than the read plus the block startup. The engine tests confirm this. This is intended: punch is the one attack you can't react to, so it is the fast poke, and you have to predict it (decided in the Phase 1 review).

## 5. Connection and whiffs

On each active frame, an attack that has not yet connected connects if the target is within range and none of these apply. When one does apply, it becomes the whiff `reason`:

| reason | condition |
|---|---|
| `out_of_range` | distance > range |
| `stale` | distance ≤ range but > the attack's stale range (see below) |
| `invulnerable` | target is knocked down |
| `ducked` | grounded high attack (`punch`, `kick`) against a target in the duck hold. Air attacks are not dodged by ducking |
| `grounded` | air-to-air attack against a target that is not airborne |
| `airborne` | overhead air attack, low attack (including landing low) or throw against an airborne target |
| `stunned` | throw against a target in hitstun |

### Stale attacks
Each fighter counts how many times in a row it has started the same attack. `jump_kick`, `dive_punch` and `low_jab` count as their own attacks, not as `kick` or `punch`. Starting a different attack resets the count to 1; moves, blocks, waits and reads don't affect it. From the 6th repeat on, the attack's range is multiplied by `max(40, 100 − 15 × (repeats − 5))` percent (integer division): 85%, 70%, 55%, then 40%. The reduced range is fixed when the attack starts and recorded in a `stale` event. Damage and frame data don't change, and there is no randomness. `in_range(a)` still reports the attack's full range, so use `repeats` to know when you're about to go stale.

### Air-attack zones
`jump_kick` and `dive_punch` take their zone from the airborne frame (1–20) on which they **start**, and keep it for their whole duration. One air attack is still allowed per jump.

| start frame | zone | hits | guarded by |
|---|---|---|---|
| 1–6 | rising overhead | grounded targets, ducking ones included. The jump continues normally | `high_block` |
| 7–13 | peak air-to-air | airborne targets only. Against a grounded target it whiffs `grounded` | — (airborne fighters can't block) |
| 14–16 | falling overhead | grounded targets, ducking ones included. Ends at touchdown | `high_block` |
| 17–20 | landing low | grounded targets. **Carries through touchdown** and finishes all its active frames on the ground (shown as `state: attack`), with landing recovery running alongside. The next statement waits for both | `low_block` |

The zone table is the config's `air_zones` and is tunable. Any hit on an airborne target knocks it down (§6).

An attack connects at most once. If its active frames end without connecting, it emits `whiff` with the reason from its last active frame.

## 6. Resolution

All connections on a frame are resolved together from the pre-resolution state.

- **Attack beats throw.** If a fighter is hit before or on its throw's active frame, the throw is abandoned. If an attack and a throw would both connect on the same frame, only the attack applies.
- **Throw tech.** If both throws connect on the same frame, there is no damage and both fighters are pushed back 2. Each emits `throw` with outcome `tech`, and both throws continue into recovery **[gap-fill]**.
- **Hit.** The target takes damage, enters hitstun (or knockdown, if the attack knocks down) and is pushed back 2.
- **Anti-air.** Any attack that hits an airborne target knocks it down and snaps its y to 0. There are no air juggles **[gap-fill]**.
- **Block.** An attack that meets a target guarding its height during the hold phase is blocked, unless it breaks the guard (below).
  - Defender: no damage, 3 frames of blockstun with the guard held, pushed back 2. The defender is not interrupted.
  - Attacker: the attack is abandoned, and the attacker is stunned for the attack's **recovery** length. It then runs `on_block` **[gap-fill]**.

  This reading of the build spec, confirmed in the Phase 1 review ("`on_block` runs after blockstun when *your* attack was blocked") keeps blocks beating attacks. A flat 3-frame stun would make a blocked attack recover faster than a whiff. A block of the wrong height does not guard. Throws ignore blocks.
- **Guard break.** Each fighter keeps a block streak: attacks guarded in a row. It resets when the fighter starts an attack, 60 frames after its last guarded attack, when it is hit or thrown, and when its guard breaks. An attack that would be the 4th guarded in a row breaks the guard instead:
  - Defender: no damage, block abandoned, pushed back 2, then 20 frames of guard-break stun (`state`/`anim` `guard_break`). It has no script input and can't guard, so attacks and throws connect as against an idle fighter. A hit during the stun starts a new combo at 1. Afterwards it runs `on_hit`.
  - Attacker: no blockstun. The attack counts as connected and runs its normal recovery, which leaves time to punish (a punch recovers in 5 frames, so a follow-up kick lands well inside the 20-frame stun).
  - A `guard_break` event is emitted instead of `block`.
- **Passivity boost.** Each fighter keeps a passive counter. Every frame it is neither stunned nor knocked down and doesn't start an attack adds 1, or 2 when it moved away from the opponent (`move_back`, or `jump_back` drift; pushback doesn't count). Starting any attack (throws, air attacks and `low_jab` included, whether or not it connects) resets it to 0. When a hit or throw lands on a target whose counter is 120 or more, its damage is multiplied by the boost: 110% at 120, plus 10 points every full 30 frames, up to 150% (at 240). The target's counter then resets to 0, and the event records `boost`. Blocks and guard breaks deal no damage and don't use the boost.
- **Throw success.** 12 damage (boosted if the target is passive) and a knockdown **[gap-fill]**.
- **Wake-up after a throw.** Throw recovery (18) is set so that an immediately repeated throw goes active on the same frame as an attack the target starts on its first actionable frame. If the first throw is active on frame k, the target wakes on k+21 and the re-throw is active on k+24, the same frame a punch started at k+21 goes active. So the triangle holds on wake-up too:
  - Attacking on wake-up beats the re-throw.
  - The re-throw beats a wake-up block, or a target that does nothing.
  - Jumping or moving away makes the re-throw whiff.

  At recovery 15 the re-throw was active on the wake-up frame itself and could not be beaten by any attack, so it was raised in the Phase 1 review.
- **Trade.** If both attacks hit on the same frame, both apply, and a `trade` event is emitted on top of the two `hit` events.
- **Pushback at a wall.** Any pushback a wall prevents is transferred to the other fighter, in the opposite direction **[gap-fill]**.

### Combos
A hit landing while the defender is still in hitstun from an earlier hit increases the defender's combo count. The count resets when the defender can act again or is knocked down.
- Damage multiplier: combos **grow** by 15 points per hit: 100, 115, 130, 145, 160%.
- Damage is `max(1, base × combo% × boost% / 10000)` with integer division, where `boost%` is the passivity boost (100 when the target isn't passive).
- The 5th hit of a combo causes a knockdown.

At the defaults (hitstun 13, pushback 2), short chains link:
- `punch kick` is a 2-hit combo.
- `punch punch kick` is 3 hits (5 + 5 + 11 damage).
- A third punch in a row whiffs `out_of_range`, because two pushbacks carry the target out of punch range. Pushback is what ends a combo.

The 5th-hit knockdown is a safety cap that the default frame data doesn't reach. (Config `2026.11-2` and earlier scaled combos down instead: ×0.8 per hit, floored at 30%.)

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
| `state` | idle, walk, attack, block, duck, airborne, landing, hitstun, blockstun, guard_break, knockdown, ko |
| `anim` | idle, check, wait, walk_forward, walk_back, punch, kick, sweep, throw, jump, jump_kick, dive_punch, duck, low_jab, high_block, low_block, landing, hitstun, blockstun, guard_break, knockdown, ko |
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
| `hit` | attacker | defender | `attack`, `damage`, `combo`, `hp` (defender's HP after), `boost` (percent, only when above 100) |
| `block` | attacker | defender | `attack` |
| `guard_break` | attacker | defender | `attack` (the attack that broke the guard; no `block` event is emitted for it) |
| `stale` | attacker | — | `attack`, `repeats` (starts in a row, 6 or more), `range` (the reduced range) |
| `whiff` | attacker | — | `attack`, `reason` (§5) |
| `throw` | thrower | defender (success only) | `attack: "throw"`, `outcome: success|tech`, plus `damage`, `hp` and (when above 100) `boost` on success |
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
