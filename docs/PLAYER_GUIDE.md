# Prompt Fighter — Player Guide

You don't press buttons in Prompt Fighter. You write your fighter's brain as a small script in the **`.pf` language**, usually with the help of an LLM, and the engine plays it frame by frame against an opponent's script. This guide teaches everything a script author needs. It describes engine config **`2026.11-2`**. The full engine reference is [`RULES.md`](RULES.md).

## 1. The round

- **Stage and fighters:** the stage runs from x = 0 to 100. p1 starts at x = 30 facing right, and p2 at x = 70 facing left. Fighters always face each other, can't pass through each other, and are never closer than 4 units.
- **HP and length:** each fighter has 100 HP. A round lasts **600 frames**.
- **Winning:** a round ends on a **KO** (HP reaches 0) or at **timeout**, where the higher HP wins and equal HP is a draw. A simultaneous KO is a draw.
- **Determinism:** there is no randomness. The same two scripts always produce the same round, and swapping sides mirrors it exactly.

## 2. Script structure

A script is a list of statements, the **main program**, plus up to two optional **handlers**:

```pf
# Lines starting with # are comments.
loop {
  if in_range(kick) {
    kick
  } else {
    move_forward x3
  }
}
on_hit { move_back x2 }
on_block { throw }
```

- Statements are separated by spaces or newlines. There is no `;`.
- Everything is lowercase.
- The main program runs from top to bottom. When it ends, the fighter stands idle, though handlers still fire.
- `on_hit` runs after you recover from being hit. `on_block` runs after your own attack was blocked (§7).
- Handlers go at the top level, at most one of each, and may not contain `loop`.

## 3. Actions

Every action uses up frames. The script waits for the action to finish before moving on.

| action | form | frames used |
|---|---|---|
| `punch`, `kick`, `sweep`, `throw` | no arguments | the attack's startup + active + recovery (§5) |
| `move_forward`, `move_back` | optional `xN`, e.g. `move_forward x3` | N frames, 1 unit per frame (default N = 1) |
| `high_block`, `low_block` | optional `hold N`, e.g. `high_block hold 6` | 1 startup + N hold + 1 recovery (default hold 10) |
| `duck` | optional `hold N` | 1 frame to start; the duck runs in the background (§6) |
| `jump`, `jump_forward`, `jump_back` | no arguments | 1 frame to start; the jump runs in the background (§6) |
| `wait N` | N is required | N frames |

Every N must be between 1 and 100. `jump_kick`, `dive_punch` and `low_jab` are **not** statements; you get them by attacking while jumping or ducking (§6).

## 4. Sensors and conditions

Conditions read **sensors**. **Every sensor read costs 2 frames**, and the value comes from the first of those 2 frames, so the information is always a little stale.

| sensor | type | meaning |
|---|---|---|
| `distance` | number | how far apart the fighters are, in units |
| `health` | number | your HP |
| `enemy.health` | number | the opponent's HP |
| `in_range(attack)` | true/false | the opponent is within that attack's range. `attack` is one of `punch`, `kick`, `sweep`, `throw`, `jump_kick`, `dive_punch`, `low_jab` |
| `enemy.airborne` | true/false | the opponent is in the air (not while landing) |
| `enemy.ducking` | true/false | the opponent is in a duck's hold phase |
| `enemy.blocking` | true/false | the opponent is holding a block (not its startup or recovery), or is in blockstun |
| `enemy.attacking` | true/false | the opponent's attack is in its startup or active phase (not recovery) |
| `enemy.knocked_down` | true/false | the opponent is on the ground after a knockdown |

**Conditions** combine these:
- Comparisons: `<`, `<=`, `>`, `>=`, `==`, `!=`, between numbers. Number literals run from 0 to 999.
- `not`, `and`, `or`, and parentheses. `and` and `or` stop early, so a sensor that is never reached costs nothing.
- A true/false sensor is used directly: write `if enemy.airborne { ... }`, not `== 1`.
- A number must be compared: write `health < 30`, not `if health`.

```pf
loop {
  if (distance <= 12 or enemy.attacking) and not enemy.knocked_down {
    high_block hold 6
  } elif health < 30 {
    move_back x10
  } else {
    move_forward x3
  }
}
```

**Control flow:**
- `if` with any number of `elif` branches and an optional `else`.
- `repeat N { ... }` runs its body N times (1–100).
- `loop { ... }` runs its body forever. It must not be empty.

Bookkeeping is free, but if a whole `loop` iteration takes no time at all, it costs 1 idle frame.

**Reaction windows.** A polling script like `loop { if enemy.attacking { high_block } }` blocks every kick and every sweep (use `low_block` for the sweep), whatever frame they start on. It can **never** react to a punch: punch's 3 startup frames are shorter than the 2-frame read plus block startup. Punch is the poke you have to predict.

## 5. Attacks and frame data

An attack goes through **startup**, then **active** (it can hit), then **recovery**. A blocked attack's recovery is spent stunned.

| attack | startup | active | recovery | height | damage | range | notes |
|---|---|---|---|---|---|---|---|
| punch | 3 | 2 | 5 | high | 5 | 8 | fastest poke |
| kick | 6 | 3 | 9 | high | 9 | 12 | longest reach |
| sweep | 8 | 3 | 14 | low | 8 | 12 | knocks down |
| throw | 5 | 1 | 18 | — | 12 | 5 | unblockable, knocks down |
| jump_kick | 3 | 5 | 0 | by zone | 10 | 10 | `kick` while airborne |
| dive_punch | 2 | 4 | 0 | by zone | 6 | 8 | `punch` while airborne |
| low_jab | 3 | 2 | 5 | low | 4 | 7 | `punch` while ducking |

- **High** attacks are blocked by `high_block` and whiff against a ducking opponent.
- **Low** attacks are blocked by `low_block` and whiff against an airborne opponent.
- Any hit on an **airborne** opponent knocks it down.

## 6. Stances: jumping and ducking

`jump`, `jump_forward`, `jump_back` and `duck` start a **stance** that runs in the background. Your script carries on with its next statement one frame later. While a stance is active:
- Sensor reads, `if`, `repeat` and `wait` keep running.
- **In the air:** the first `kick` becomes `jump_kick` and the first `punch` becomes `dive_punch`. You get **one air attack per jump**.
- **In a duck's hold phase:** `punch` becomes `low_jab`, as often as you like.
- **Anything else waits:** every other action waits until the stance has completely ended. That includes a second air attack.

A jump lasts **20 airborne frames** (10 rising, 10 falling) plus **2 landing frames**. `jump_forward` and `jump_back` move 1 unit per airborne frame. A duck is 2 startup frames + hold (default 10) + 2 recovery frames. Ducking dodges grounded high attacks (punch, kick) but **not** air attacks.

```pf
jump_forward
kick
duck hold 20
punch
```

### Air-attack zones

What an air attack can hit depends on the **airborne frame it starts on** (frame 1 is the frame you left the ground). The zone is fixed when the attack starts.

| starts on airborne frame | zone | hits | blocked by |
|---|---|---|---|
| 1–6 | rising overhead | grounded opponents, ducking ones included | `high_block` |
| 7–13 | peak air-to-air | **airborne opponents only**; whiffs against a grounded one (`grounded`) | — |
| 14–16 | falling overhead | grounded opponents, ducking ones included | `high_block` |
| 17–20 | landing low | grounded opponents; **keeps going after you land** and finishes on the ground | `low_block` |

Count frames with `wait`. The jump starts on airborne frame 1, so the next statement runs on frame 2:
- `jump_forward kick` is a rising overhead (starts on frame 2).
- `jump wait 7 kick` is air-to-air (starts on frame 9).
- `jump wait 15 kick` is a landing low (starts on frame 17).

```pf
loop {
  if distance > 24 {
    move_forward x5
  } else {
    jump_forward
    wait 15
    kick
  }
}
```

## 7. Hitting, blocking, throwing

- **Hit:** damage, then **13 frames of hitstun** (no input), and the target is pushed back 2 units. Pushback a wall would absorb moves the attacker away instead.
- **Block:** the defender takes no damage, gets 3 frames of blockstun while still guarding, and is pushed back 2. The **attacker** is stunned for its attack's recovery and then runs `on_block`. A block of the wrong height does nothing, and only the block's hold phase guards.
- **Throw:** ignores blocks, deals 12 damage and knocks the target down.
- **The triangle:** attacks beat throws (a thrower hit before or on its active frame loses its throw), throws beat blocks, and blocks beat attacks.
  - **Two throws at once:** if both fighters throw on the same frame, they *tech*: no damage, and both are pushed apart.
  - **The triangle holds on wake-up too.** If you throw and immediately throw again, your second throw goes active on the same frame as an attack the target starts the moment it stands up, and the attack wins. A wake-up block, or doing nothing, gets thrown again.
- **Knockdown:** 20 frames on the ground, invulnerable and with no input. Caused by a sweep, a throw, any hit on an airborne fighter, or the 5th hit of a combo.
- **Trade:** if both attacks hit on the same frame, both take damage.

### Combos

A hit that lands while the opponent is still in hitstun extends a **combo**. Each extra hit's damage is scaled: ×80% per hit, never below 30% (100, 80, 64, 51, 40…). The 5th hit knocks down. At the defaults, short chains link:

```pf
punch
punch
kick
```

From distance 6 that's a 3-hit combo for 5 + 4 + 5 damage. A third punch in a row misses, because two pushbacks carry the opponent out of punch range.

## 8. Interrupts and handlers

When you are hit (hitstun or knockdown), your current action and stance are abandoned and your script freezes. When you can act again:
1. `on_hit` runs, if you have one.
2. The script continues at the **statement after** the one that was interrupted.
   - Interrupted during an `if` condition? The whole `if` chain is skipped.
   - Interrupted on the last statement of a `repeat` or `loop`? The next iteration starts.

When your attack is blocked, you are stunned for the attack's recovery, then `on_block` runs, then the script continues after the attack. If you're hit again during a handler, the handler starts over after the new stun. Waking up from a knockdown counts as being hit, so `on_hit` runs.

## 9. Limits and common errors

- **Limits:** a script is at most **4096 bytes**, with at most **40 statements** (each action, `if`, `elif`, `repeat` and `loop` counts; `else` and handlers don't) and a **nesting depth 4** (blocks inside blocks, handler blocks included).
- **Validation:** reports up to 10 errors as `line:col: message`, usually with a fix.

| you wrote | write instead |
|---|---|
| `kick x2` | `repeat 2 { kick }` |
| `wait` | `wait 4` (needs a frame count) |
| `move_back 4` | `move_back x4` |
| `high_block x3` | `high_block hold 3` |
| `jump_kick` | `jump_forward kick` |
| `if health { ... }` | `if health < 30 { ... }` |
| `if enemy.airborne == 1 { ... }` | `if enemy.airborne { ... }` |
| `else if` | `elif` |
| `while`, `for` | `loop { ... }`, `repeat N { ... }` |
| `&&`, `\|\|`, `!` | `and`, `or`, `not` |
| `kick;` | `kick` (no semicolons) |

```text
kik          → 1:1: unknown action "kik" — did you mean "kick"?
```

## 10. Strategy notes

These come from the six sample bots (`pf selfplay scripts/` at config 2026.11-2). Every bot beats at least one other and loses to at least one other:

| bot | idea | beats | loses to |
|---|---|---|---|
| `turtle` | holds `high_block`, punches when close, throws when its own attack is blocked | jumper, thrower | lazy, rushdown |
| `rushdown` | walks in and kicks, sweeps blockers | lazy, thrower, turtle | jumper, ducker |
| `jumper` | jumps in with landing-low kicks | ducker, lazy, rushdown, thrower | turtle |
| `thrower` | walks in and throws | lazy | everyone else |
| `ducker` | ducks under high attacks, jabs and sweeps | lazy, rushdown, thrower | jumper |
| `lazy` | a fixed routine with no sensor reads | turtle | everyone else |

- **Sensors cost time.** A script that checks three sensors before acting is 6 frames late. Check the one that matters, or don't check at all, as `lazy` shows.
- **Punch is unreactable.** Use it to interrupt throws and slow attacks.
- **Ducking beats high pokes** but loses to overheads. **Jumping beats ducking and throws** but loses to anti-air hits, because any hit on a jumper knocks it down.
- **Blocking is strong:** a blocked kick leaves the attacker stunned for 9 frames, so throw or attack straight after a block.
- **Long `hold` values** leave you committed. Short blocks inside a `loop` re-check more often.

## 11. How to play

**With an LLM, over MCP.** Run `pf mcp`; your MCP client launches it. It gives the model these tools:
- `get_player_guide`: this document.
- `validate_script`.
- `simulate_round`: spar against a script or a sample bot from `list_sample_bots`.
- For matches on a `pf serve` server: `create_match`, `join_match`, `submit_script`, `get_match` and `get_round_replay`.

Claude Code: `claude mcp add prompt-fighter -- pf mcp`. Claude Desktop (`claude_desktop_config.json`):

```json
{ "mcpServers": { "prompt-fighter": { "command": "pf", "args": ["mcp"] } } }
```

**On the command line:**
- `pf validate my.pf`
- `pf sim my.pf scripts/turtle.pf` (add `-v` to see sensor reads, or `--json` for the full log)
- `pf selfplay scripts/`

**Matches over HTTP:** `pf serve` hosts best-of-3 matches. You and your opponent each submit a script before every round, so you can revise between rounds:
- The first to win 2 rounds wins, and drawn rounds don't count.
- A match lasts at most 5 rounds.
- Sides alternate each round.

See [`API.md`](API.md).
