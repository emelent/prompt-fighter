package sim

import "fmt"

// EventLines renders the events as the human-readable lines printed by
// "pf sim": "[f NNN] <actor> <description>". Check events are included only
// when includeChecks is true.
func (l *Log) EventLines(includeChecks bool) []string {
	lines := make([]string, 0, len(l.Events))
	for _, e := range l.Events {
		if e.Type == "check" && !includeChecks {
			continue
		}
		lines = append(lines, fmt.Sprintf("[f %03d] %s %s", e.Frame, e.Actor, describeEvent(e)))
	}
	return lines
}

// ResultLine renders the round result as printed at the end of "pf sim".
func (l *Log) ResultLine() string {
	r := l.Result
	hp := fmt.Sprintf("hp %d-%d", r.HP[0], r.HP[1])
	if r.Winner == nil {
		return fmt.Sprintf("result: draw by %s at f%d (%s)", r.Reason, r.EndFrame, hp)
	}
	return fmt.Sprintf("result: %s wins by %s at f%d (%s)", *r.Winner, r.Reason, r.EndFrame, hp)
}

func describeEvent(e Event) string {
	d := e.Detail
	switch e.Type {
	case "hit":
		return fmt.Sprintf("%s hits %s for %d (combo %d, hp %d)", d.Attack, e.Target, *d.Damage, d.Combo, *d.HP)
	case "block":
		return fmt.Sprintf("%s blocked by %s", d.Attack, e.Target)
	case "whiff":
		return fmt.Sprintf("%s whiffs (%s)", d.Attack, d.Reason)
	case "throw":
		if d.Outcome == "tech" {
			return "throw teched"
		}
		return fmt.Sprintf("throws %s for %d (hp %d)", e.Target, *d.Damage, *d.HP)
	case "trade":
		return fmt.Sprintf("trades %s for %s (%d/%d)", d.Attacks[0], d.Attacks[1], d.Damages[0], d.Damages[1])
	case "knockdown":
		return fmt.Sprintf("knocked down (%s)", d.Reason)
	case "ko":
		return "KO"
	case "check":
		return fmt.Sprintf("checks %s = %s (line %d)", d.Sensor, string(d.Value), d.Line)
	case "script_end":
		return "script ended"
	case "handler_enter":
		return "enters " + d.Handler
	}
	return e.Type
}
