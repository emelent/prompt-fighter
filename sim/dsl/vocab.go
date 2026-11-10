// Package dsl implements the .pf fight-script language: lexer, parser and
// validator. It knows the language vocabulary but no game rules.
package dsl

// Action is a statement-level action keyword.
type Action int

const (
	ActPunch Action = iota
	ActKick
	ActSweep
	ActThrow
	ActJump
	ActJumpForward
	ActJumpBack
	ActDuck
	ActHighBlock
	ActLowBlock
	ActMoveForward
	ActMoveBack
	ActWait
	NumActions
)

var actionNames = [NumActions]string{
	"punch", "kick", "sweep", "throw",
	"jump", "jump_forward", "jump_back", "duck",
	"high_block", "low_block", "move_forward", "move_back", "wait",
}

func (a Action) String() string { return actionNames[a] }

// ArgForm describes which argument an action accepts.
type ArgForm int

const (
	ArgNone  ArgForm = iota // no arguments
	ArgHold                 // optional "hold N"
	ArgCount                // optional "xN"
	ArgWait                 // required "N"
)

// Arg returns the argument form for the action.
func (a Action) Arg() ArgForm {
	switch a {
	case ActDuck, ActHighBlock, ActLowBlock:
		return ArgHold
	case ActMoveForward, ActMoveBack:
		return ArgCount
	case ActWait:
		return ArgWait
	}
	return ArgNone
}

// IsStance reports whether the action starts a background stance.
func (a Action) IsStance() bool {
	return a == ActJump || a == ActJumpForward || a == ActJumpBack || a == ActDuck
}

// ActionByName looks up an action keyword.
func ActionByName(s string) (Action, bool) {
	for i, n := range actionNames {
		if n == s {
			return Action(i), true
		}
	}
	return 0, false
}

// Attack identifies an attack, including overlay attacks.
type Attack int

const (
	AtkPunch Attack = iota
	AtkKick
	AtkSweep
	AtkThrow
	AtkJumpKick
	AtkDivePunch
	AtkLowJab
	NumAttacks
)

var attackNames = [NumAttacks]string{"punch", "kick", "sweep", "throw", "jump_kick", "dive_punch", "low_jab"}

func (a Attack) String() string { return attackNames[a] }

// AttackNames returns attack names in canonical order.
func AttackNames() []string { return attackNames[:] }

// AttackByName looks up an attack name.
func AttackByName(s string) (Attack, bool) {
	for i, n := range attackNames {
		if n == s {
			return Attack(i), true
		}
	}
	return 0, false
}

// Sensor is a condition input read from the game state.
type Sensor int

const (
	SenDistance Sensor = iota
	SenHealth
	SenEnemyHealth
	SenGuard        // own block streak
	SenPassive      // own passive frames
	SenEnemyPassive // enemy passive frames
	SenRepeats      // starts of own last attack in a row
	SenEnemyAirborne
	SenEnemyDucking
	SenEnemyBlocking
	SenEnemyAttacking
	SenEnemyKnockedDown
	SenInRange
	NumSensors
)

var sensorNames = [NumSensors]string{
	"distance", "health", "enemy.health", "guard", "passive", "enemy.passive", "repeats",
	"enemy.airborne", "enemy.ducking", "enemy.blocking", "enemy.attacking", "enemy.knocked_down",
	"in_range",
}

func (s Sensor) String() string { return sensorNames[s] }

// IsBool reports whether the sensor yields true/false (otherwise an integer).
func (s Sensor) IsBool() bool { return s >= SenEnemyAirborne }

// SensorByName looks up a sensor name.
func SensorByName(s string) (Sensor, bool) {
	for i, n := range sensorNames {
		if n == s {
			return Sensor(i), true
		}
	}
	return 0, false
}

var stmtKeywords = []string{"if", "repeat", "loop", "on_hit", "on_block"}

// Hand-written aliases for likely LLM mistakes, checked before edit distance.
var actionAliases = map[string]string{
	"block": "high_block", "guard": "high_block", "jab": "punch", "hit": "punch",
	"crouch": "duck", "walk_forward": "move_forward", "forward": "move_forward",
	"advance": "move_forward", "approach": "move_forward", "move": "move_forward",
	"walk_back": "move_back", "backward": "move_back", "retreat": "move_back",
	"back": "move_back", "grab": "throw", "idle": "wait", "sleep": "wait",
	"low_kick": "sweep", "trip": "sweep", "hop": "jump",
}

var sensorAliases = map[string]string{
	"enemy.jumping": "enemy.airborne", "enemy.in_air": "enemy.airborne", "enemy.air": "enemy.airborne",
	"enemy.crouching": "enemy.ducking", "enemy.hp": "enemy.health", "hp": "health",
	"my.health": "health", "self.health": "health", "me.health": "health", "my.hp": "health",
	"enemy.distance": "distance", "range": "distance", "dist": "distance",
	"enemy.down": "enemy.knocked_down", "enemy.knockdown": "enemy.knocked_down",
	"enemy.guarding": "enemy.blocking", "enemy.attack": "enemy.attacking",
	"my.passive": "passive", "enemy.idle": "enemy.passive", "streak": "guard", "block_streak": "guard",
	"repeat_count": "repeats", "stale": "repeats",
}

// suggest returns the best suggestion for word among candidates (alias table
// first, then edit distance <= 2, ties broken by candidate order).
func suggest(word string, aliases map[string]string, candidates ...[]string) string {
	if s, ok := aliases[word]; ok {
		return s
	}
	best, bestD := "", 3
	for _, list := range candidates {
		for _, c := range list {
			if d := levenshtein(word, c); d < bestD {
				best, bestD = c, d
			}
		}
	}
	return best
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
