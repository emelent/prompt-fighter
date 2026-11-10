package dsl

import (
	"strings"
	"testing"
)

func TestValidateMessages(t *testing.T) {
	tests := []struct {
		name, src string
		want      []string // exact `line:col: message` list; nil means valid
	}{
		// valid
		{"build spec example", "loop {\n  if in_range(sweep) and not enemy.airborne { sweep }\n  elif enemy.airborne { high_block hold 6 }\n  elif health < 30 { move_back x10 }\n  else { move_forward x3 }\n}\non_hit { move_back x2 }\non_block { throw }\n", nil},
		{"valid arg forms", "move_forward x3 move_back high_block hold 6 duck wait 4 jump_forward", nil},
		{"valid compound condition", "if (distance <= 12 or enemy.attacking) and not enemy.knocked_down { kick }", nil},
		{"nesting at limit", "loop { if distance < 10 { repeat 2 { if enemy.ducking { sweep } } } }", nil},
		{"empty script", "", nil},

		// structure
		{"duplicate handler", "on_hit { kick }\non_hit { punch }", []string{"2:1: only one `on_hit` handler is allowed"}},
		{"handler not top level", "loop { on_block { throw } }", []string{"1:8: handlers must be declared at the top level of the script, not inside a block"}},

		// actions
		{"kick x2", "kick x2", []string{"1:6: `kick` takes no count — use `repeat 2 { kick }`"}},
		{"wait missing", "wait", []string{"1:1: `wait` requires a frame count such as `wait 4`"}},
		{"count zero", "move_forward x0", []string{"1:14: `move_forward` count must be between 1 and 100 (got 0)"}},
		{"count too big", "wait 500", []string{"1:6: `wait` count must be between 1 and 100 (got 500)"}},
		{"repeat count", "repeat 101 { kick }", []string{"1:8: `repeat` count must be between 1 and 100 (got 101)"}},
		{"jump_kick statement", "jump_kick", []string{"1:1: `jump_kick` is not a statement — it happens when you `kick` while airborne, e.g. `jump_forward kick`"}},
		{"low_jab statement", "low_jab", []string{"1:1: `low_jab` is not a statement — it happens when you `punch` while ducking, e.g. `duck hold 10 punch`"}},
		{"block with x", "high_block x3", []string{"1:12: `high_block` takes `hold N`, not `xN` — e.g. `high_block hold 3`"}},
		{"move without x", "move_back 4", []string{"1:11: write `move_back x4` — the count needs an `x` in front"}},
		{"hold on attack", "kick hold 3", []string{"1:6: `kick` does not take `hold`; only `duck`, `high_block` and `low_block` do"}},
		{"misspelled action", "punch\n\n    kik", []string{`3:5: unknown action "kik" — did you mean "kick"?`}},
		{"alias action", "block hold 3", []string{`1:1: unknown action "block" — did you mean "high_block"?`}},

		// control flow
		{"empty loop", "loop { }", []string{"1:1: a `loop` body must contain at least one statement"}},
		{"loop in handler", "on_hit { loop { punch } }", []string{"1:10: `loop` is not allowed inside handlers because a handler must finish so the main script can resume"}},
		{"dangling elif", "elif health < 30 { kick }", []string{"1:1: `elif` must follow an `if` or `elif` block"}},
		{"else if", "if health < 5 { kick } else if health < 9 { punch }", []string{"1:29: use `elif` instead of `else if`"}},

		// conditions
		{"int as bool", "if health { kick }", []string{"1:4: `health` is a number and must be compared, e.g. `health < 30`"}},
		{"bool compared", "if enemy.airborne == 1 { kick }", []string{"1:4: `enemy.airborne` is true/false and should be used directly, e.g. `if enemy.airborne { ... }`"}},
		{"unknown sensor alias", "if enemy.jumping { kick }", []string{`1:4: unknown sensor "enemy.jumping" — did you mean "enemy.airborne"?`}},
		{"unknown sensor edit distance", "if enemy.helth < 3 { kick }", []string{`1:4: unknown sensor "enemy.helth" — did you mean "enemy.health"?`}},
		{"in_range bad arg", "if in_range(jump) { kick }", []string{"1:13: `in_range` needs an attack name: punch, kick, sweep, throw, jump_kick, dive_punch, low_jab"}},
		{"attack as condition", "if kick { kick }", []string{`1:4: "kick" is an attack, not a condition — did you mean ` + "`in_range(kick)`?"}},
		{"literal too big", "if distance < 1000 { kick }", []string{"1:15: number 1000 is too large; numbers in conditions must be between 0 and 999"}},
		{"number as condition", "if 5 { kick }", []string{"1:4: a number cannot be used as a condition; compare a sensor to it, e.g. `distance < 5`"}},
		{"missing condition", "if { kick }", []string{"1:4: missing condition before `{`"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.src, DefaultLimits())
			var got []string
			for _, e := range errs {
				got = append(got, e.String())
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestValidateLimits(t *testing.T) {
	t.Run("41 statements", func(t *testing.T) {
		src := strings.Repeat("punch\n", 41)
		errs := Validate(src, DefaultLimits())
		want := "41:1: script has 41 statements; the maximum is 40 — use `repeat` or `loop` to shorten it"
		if len(errs) != 1 || errs[0].String() != want {
			t.Errorf("got %v, want %s", errs, want)
		}
	})
	t.Run("40 statements ok", func(t *testing.T) {
		if errs := Validate(strings.Repeat("punch\n", 40), DefaultLimits()); len(errs) != 0 {
			t.Errorf("got %v", errs)
		}
	})
	t.Run("handler statements count", func(t *testing.T) {
		src := strings.Repeat("punch\n", 39) + "on_hit { kick kick }"
		if errs := Validate(src, DefaultLimits()); len(errs) != 1 || !strings.Contains(errs[0].Msg, "41 statements") {
			t.Errorf("got %v", errs)
		}
	})
	t.Run("if elif count, else does not", func(t *testing.T) {
		// 1 if + 1 elif + 3 actions = 5
		src := strings.Repeat("punch\n", 35) + "if health < 5 { kick } elif health < 9 { kick } else { kick }"
		if errs := Validate(src, DefaultLimits()); len(errs) != 0 {
			t.Errorf("40 statements should be valid, got %v", errs)
		}
		src = strings.Repeat("punch\n", 36) + "if health < 5 { kick } elif health < 9 { kick } else { kick }"
		if errs := Validate(src, DefaultLimits()); len(errs) != 1 {
			t.Errorf("41 statements should be invalid, got %v", errs)
		}
	})
	t.Run("too deep", func(t *testing.T) {
		src := "loop { repeat 2 { repeat 2 { repeat 2 { repeat 2 { kick } } } } }"
		want := "1:50: blocks are nested too deeply (depth 5); the maximum nesting depth is 4 — flatten the script"
		errs := Validate(src, DefaultLimits())
		if len(errs) != 1 || errs[0].String() != want {
			t.Errorf("got %v, want %s", errs, want)
		}
	})
	t.Run("handler body counts as a block", func(t *testing.T) {
		src := "on_hit { repeat 2 { repeat 2 { repeat 2 { repeat 2 { kick } } } } }"
		if errs := Validate(src, DefaultLimits()); len(errs) != 1 || !strings.Contains(errs[0].Msg, "depth 5") {
			t.Errorf("got %v", errs)
		}
	})
	t.Run("oversized", func(t *testing.T) {
		src := strings.Repeat("#", 4097)
		want := "1:1: script is 4097 bytes; the maximum is 4096 bytes — shorten it (remove comments, use `loop`/`repeat`)"
		errs := Validate(src, DefaultLimits())
		if len(errs) != 1 || errs[0].String() != want {
			t.Errorf("got %v", errs)
		}
		if errs := Validate(strings.Repeat("#", 4096), DefaultLimits()); len(errs) != 0 {
			t.Errorf("4096 bytes should be valid, got %v", errs)
		}
	})
}

func TestValidateValidReturnsEmpty(t *testing.T) {
	if errs := Validate("kick", DefaultLimits()); errs == nil || len(errs) != 0 {
		if len(errs) != 0 {
			t.Errorf("got %v", errs)
		}
	}
}
