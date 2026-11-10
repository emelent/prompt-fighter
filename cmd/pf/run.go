package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"promptfighter/sim"
	"promptfighter/sim/dsl"
)

const usage = `usage:
  pf validate <file>...
  pf sim <a.pf> <b.pf> [--json] [--frames N] [-v]
  pf selfplay <dir>`

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	switch args[0] {
	case "validate":
		return cmdValidate(args[1:], stdout, stderr)
	case "sim":
		return cmdSim(args[1:], stdout, stderr)
	case "selfplay":
		return cmdSelfplay(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stdout, usage)
		return 0
	}
	fmt.Fprintf(stderr, "pf: unknown command %q\n%s\n", args[0], usage)
	return 2
}

func printErrors(w io.Writer, file string, errs []dsl.Error) {
	for _, e := range errs {
		fmt.Fprintf(w, "%s:%s\n", file, e.String())
	}
}

// --- validate ---

func cmdValidate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: pf validate <file>...")
		return 2
	}
	lim := sim.DefaultConfig().Limits
	code := 0
	for _, path := range args {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "pf: %v\n", err)
			return 2
		}
		if errs := dsl.Validate(string(src), lim); len(errs) > 0 {
			printErrors(stdout, path, errs)
			code = 1
		} else {
			fmt.Fprintf(stdout, "%s: ok\n", path)
		}
	}
	return code
}

// --- sim ---

func cmdSim(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sim", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "print the event log JSON")
	verbose := fs.Bool("v", false, "include check events in the text log")
	cfg := sim.DefaultConfig()
	nframes := fs.Int("frames", cfg.FrameBudget, "frame budget (>= 1)")
	// Allow flags before and after the two file arguments.
	var files []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		if fs.NArg() == 0 {
			break
		}
		files = append(files, fs.Arg(0))
		rest = fs.Args()[1:]
	}
	if len(files) != 2 {
		fmt.Fprintln(stderr, "usage: pf sim <a.pf> <b.pf> [--json] [--frames N] [-v]")
		return 2
	}
	if *nframes < 1 {
		fmt.Fprintf(stderr, "pf: --frames must be >= 1, got %d\n", *nframes)
		return 2
	}
	cfg.FrameBudget = *nframes
	var ents [2]sim.Entrant
	for i, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "pf: %v\n", err)
			return 2
		}
		ents[i] = sim.Entrant{Name: filepath.Base(path), Source: string(src)}
	}
	// Report validation errors for both scripts, like pf validate.
	invalid := false
	for i, path := range files {
		if errs := dsl.Validate(ents[i].Source, cfg.Limits); len(errs) > 0 {
			printErrors(stdout, path, errs)
			invalid = true
		}
	}
	if invalid {
		return 1
	}
	l, err := sim.Simulate(cfg, ents[0], ents[1])
	if err != nil {
		fmt.Fprintf(stderr, "pf: %v\n", err)
		return 1
	}
	if *asJSON {
		stdout.Write(l.JSON())
		return 0
	}
	writeText(stdout, l, *verbose)
	return 0
}

func describe(e sim.Event) string {
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

func writeText(w io.Writer, l *sim.Log, verbose bool) {
	for _, e := range l.Events {
		if e.Type == "check" && !verbose {
			continue
		}
		fmt.Fprintf(w, "[f %03d] %s %s\n", e.Frame, e.Actor, describe(e))
	}
	r := l.Result
	hp := fmt.Sprintf("hp %d-%d", r.HP[0], r.HP[1])
	if r.Winner == nil {
		fmt.Fprintf(w, "result: draw by %s at f%d (%s)\n", r.Reason, r.EndFrame, hp)
	} else {
		fmt.Fprintf(w, "result: %s wins by %s at f%d (%s)\n", *r.Winner, r.Reason, r.EndFrame, hp)
	}
}

// --- selfplay ---

type record struct{ w, d, l int }

func cmdSelfplay(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: pf selfplay <dir>")
		return 2
	}
	names, matrix, err := selfplay(args[0], stdout)
	if err != nil {
		fmt.Fprintf(stderr, "pf: %v\n", err)
		return 2
	}
	writeMatrix(stdout, names, matrix)
	return 0
}

// selfplay loads every *.pf in dir (sorted), reports invalid ones to out and
// plays every ordered pair of valid scripts. matrix[i][j] is i's record vs j
// across both side assignments.
func selfplay(dir string, out io.Writer) ([]string, [][]record, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.pf"))
	if err != nil {
		return nil, nil, err
	}
	if len(paths) == 0 {
		return nil, nil, errors.New("no .pf files in " + dir)
	}
	sort.Strings(paths)
	cfg := sim.DefaultConfig()
	var names []string
	var ents []sim.Entrant
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		if errs := dsl.Validate(string(src), cfg.Limits); len(errs) > 0 {
			fmt.Fprintf(out, "excluded %s (invalid):\n", path)
			printErrors(out, path, errs)
			continue
		}
		name := strings.TrimSuffix(filepath.Base(path), ".pf")
		names = append(names, name)
		ents = append(ents, sim.Entrant{Name: filepath.Base(path), Source: string(src)})
	}
	n := len(ents)
	matrix := make([][]record, n)
	for i := range matrix {
		matrix[i] = make([]record, n)
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			l, err := sim.Simulate(cfg, ents[i], ents[j])
			if err != nil {
				return nil, nil, err
			}
			switch w := l.Result.Winner; {
			case w == nil:
				matrix[i][j].d++
				matrix[j][i].d++
			case *w == "p1":
				matrix[i][j].w++
				matrix[j][i].l++
			default:
				matrix[i][j].l++
				matrix[j][i].w++
			}
		}
	}
	return names, matrix, nil
}

func writeMatrix(w io.Writer, names []string, m [][]record) {
	width := len("total")
	for _, n := range names {
		width = max(width, len(n))
	}
	cell := func(s string) string { return s + strings.Repeat(" ", width+2-utf8.RuneCountInString(s)) }
	var b strings.Builder
	b.WriteString(cell(""))
	for _, n := range names {
		b.WriteString(cell(n))
	}
	b.WriteString("total\n")
	for i, n := range names {
		b.WriteString(cell(n))
		var tot record
		for j := range names {
			if i == j {
				b.WriteString(cell("—"))
				continue
			}
			r := m[i][j]
			tot.w, tot.d, tot.l = tot.w+r.w, tot.d+r.d, tot.l+r.l
			b.WriteString(cell(fmt.Sprintf("%d-%d-%d", r.w, r.d, r.l)))
		}
		b.WriteString(fmt.Sprintf("%d-%d-%d\n", tot.w, tot.d, tot.l))
	}
	io.WriteString(w, b.String())
}
