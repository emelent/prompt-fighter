// Command nextversion prints the next semantic version for a release, derived
// from conventional commits since the latest v* tag (nothing when there are
// no new commits). See Next for the rules.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Commit is one commit message.
type Commit struct {
	Subject, Body string
}

var (
	header   = regexp.MustCompile(`^(\w+)(\([^)]*\))?(!)?:`)
	breaking = regexp.MustCompile(`(?m)^BREAKING[ -]CHANGE:`)
	tagRE    = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
)

// Next returns the version after lastTag ("" for none) given the commits made
// since it, and false when there is nothing to release:
//   - a breaking change (type! or a BREAKING CHANGE footer) bumps major, or
//     minor while the major version is 0
//   - otherwise a feat bumps minor
//   - otherwise any commit bumps patch
//
// With no tag yet, the first version is v0.1.0.
func Next(lastTag string, commits []Commit) (string, bool, error) {
	if len(commits) == 0 {
		return "", false, nil
	}
	if lastTag == "" {
		return "v0.1.0", true, nil
	}
	m := tagRE.FindStringSubmatch(lastTag)
	if m == nil {
		return "", false, fmt.Errorf("latest tag %q is not vMAJOR.MINOR.PATCH", lastTag)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	isBreaking, isFeat := false, false
	for _, c := range commits {
		h := header.FindStringSubmatch(strings.TrimSpace(c.Subject))
		if h != nil && h[3] == "!" || breaking.MatchString(c.Body) {
			isBreaking = true
		}
		if h != nil && strings.EqualFold(h[1], "feat") {
			isFeat = true
		}
	}
	switch {
	case isBreaking && major > 0:
		major, minor, patch = major+1, 0, 0
	case isBreaking || isFeat:
		minor, patch = minor+1, 0
	default:
		patch++
	}
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch), true, nil
}

func git(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return string(out), err
}

func main() {
	tag, err := git("describe", "--tags", "--abbrev=0", "--match", "v[0-9]*")
	tag = strings.TrimSpace(tag)
	if err != nil {
		tag = "" // no release tag yet
	}
	rng := "HEAD"
	if tag != "" {
		rng = tag + "..HEAD"
	}
	log, err := git("log", rng, "--format=%s%x1f%b%x1e")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nextversion: git log:", err)
		os.Exit(1)
	}
	var commits []Commit
	for _, rec := range strings.Split(log, "\x1e") {
		rec = strings.TrimLeft(rec, "\n")
		if rec == "" {
			continue
		}
		subject, body, _ := strings.Cut(rec, "\x1f")
		commits = append(commits, Commit{Subject: subject, Body: body})
	}
	next, ok, err := Next(tag, commits)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nextversion:", err)
		os.Exit(1)
	}
	if ok {
		fmt.Println(next)
	}
}
