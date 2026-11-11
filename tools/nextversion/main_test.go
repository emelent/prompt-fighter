package main

import "testing"

func TestNext(t *testing.T) {
	c := func(subject string, body ...string) Commit {
		b := ""
		if len(body) > 0 {
			b = body[0]
		}
		return Commit{Subject: subject, Body: b}
	}
	tests := []struct {
		name, last string
		commits    []Commit
		want       string // "" = nothing to release
	}{
		{"feature and fix", "v0.4.2", []Commit{c("fix: x"), c("feat(web): y")}, "v0.5.0"},
		{"breaking before 1.0", "v0.4.2", []Commit{c("feat!: new protocol")}, "v0.5.0"},
		{"breaking after 1.0 (footer)", "v1.4.2", []Commit{c("refactor: join", "Details.\n\nBREAKING CHANGE: removed join script")}, "v2.0.0"},
		{"breaking after 1.0 (bang with scope)", "v1.4.2", []Commit{c("fix(server)!: drop old field")}, "v2.0.0"},
		{"breaking hyphen footer", "v1.0.0", []Commit{c("chore: x", "BREAKING-CHANGE: y")}, "v2.0.0"},
		{"fix only", "v0.4.2", []Commit{c("fix: x")}, "v0.4.3"},
		{"non-conventional subject", "v0.4.2", []Commit{c("update ui")}, "v0.4.3"},
		{"merge commit", "v0.4.2", []Commit{c("Merge pull request #3 from x/y")}, "v0.4.3"},
		{"feat in caps", "v0.4.2", []Commit{c("FEAT: shout")}, "v0.5.0"},
		{"feat-like word is not feat", "v0.4.2", []Commit{c("feature: not a type")}, "v0.4.3"},
		{"BREAKING mid-line is not a footer", "v1.0.0", []Commit{c("docs: note", "this is not a BREAKING CHANGE: really")}, "v1.0.1"},
		{"first release", "", []Commit{c("update ui")}, "v0.1.0"},
		{"nothing new", "v0.4.2", nil, ""},
	}
	for _, tt := range tests {
		got, ok, err := Next(tt.last, tt.commits)
		if err != nil {
			t.Errorf("%s: error %v", tt.name, err)
			continue
		}
		if (tt.want == "") == ok || got != tt.want {
			t.Errorf("%s: got %q (ok=%v), want %q", tt.name, got, ok, tt.want)
		}
	}
	if _, _, err := Next("release-1", []Commit{c("fix: x")}); err == nil {
		t.Error("a malformed tag should be an error")
	}
}
