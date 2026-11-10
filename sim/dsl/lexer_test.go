package dsl

import (
	"fmt"
	"strings"
	"testing"
)

func tokStr(toks []Token) string {
	var parts []string
	for _, t := range toks {
		k := map[TokKind]string{TEOF: "EOF", TIdent: "ID", TInt: "INT", TCount: "CNT", TLBrace: "{", TRBrace: "}", TLParen: "(", TRParen: ")", TCmp: "CMP"}[t.Kind]
		parts = append(parts, fmt.Sprintf("%s:%s@%d:%d", k, t.Text, t.Pos.Line, t.Pos.Col))
	}
	return strings.Join(parts, " ")
}

func TestLexTokens(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{"action with count", "move_forward x3", "ID:move_forward@1:1 CNT:x3@1:14 EOF:end of script@1:16"},
		{"x alone is ident", "x", "ID:x@1:1 EOF:end of script@1:2"},
		{"x with letters is ident", "x3a", "ID:x3a@1:1 EOF:end of script@1:4"},
		{"dotted sensor", "enemy.airborne", "ID:enemy.airborne@1:1 EOF:end of script@1:15"},
		{"comparison ops", "a<=1 b>=2 c==3 d!=4 e<5 f>6",
			"ID:a@1:1 CMP:<=@1:2 INT:1@1:4 ID:b@1:6 CMP:>=@1:7 INT:2@1:9 ID:c@1:11 CMP:==@1:12 INT:3@1:14 ID:d@1:16 CMP:!=@1:17 INT:4@1:19 ID:e@1:21 CMP:<@1:22 INT:5@1:23 ID:f@1:25 CMP:>@1:26 INT:6@1:27 EOF:end of script@1:28"},
		{"braces parens", "in_range(kick) { }", "ID:in_range@1:1 (:(@1:9 ID:kick@1:10 ):)@1:14 {:{@1:16 }:}@1:18 EOF:end of script@1:19"},
		{"comments and lines", "# hi\n  kick # trailing\npunch", "ID:kick@2:3 ID:punch@3:1 EOF:end of script@3:6"},
		{"hold", "duck hold 6", "ID:duck@1:1 ID:hold@1:6 INT:6@1:11 EOF:end of script@1:12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, errs := Lex(tt.src)
			if len(errs) != 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if got := tokStr(toks); got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestLexForeignOperatorHints(t *testing.T) {
	tests := []struct {
		src, wantErr, wantTok string
	}{
		{"a && b", "1:3: use `and` instead of `&&`", "ID:and@1:3"},
		{"a || b", "1:3: use `or` instead of `||`", "ID:or@1:3"},
		{"!a", "1:1: use `not` instead of `!`", "ID:not@1:1"},
		{"kick;", "1:5: `;` is not needed; separate statements with spaces or newlines", ""},
		{"a = 1", "1:3: use `==` to compare, not `=`", "CMP:==@1:3"},
		{"kick @", "1:6: unexpected character '@'", ""},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			toks, errs := Lex(tt.src)
			if len(errs) != 1 || errs[0].String() != tt.wantErr {
				t.Fatalf("errors = %v, want [%s]", errs, tt.wantErr)
			}
			if tt.wantTok != "" && !strings.Contains(tokStr(toks), tt.wantTok) {
				t.Errorf("tokens %s missing replacement %s", tokStr(toks), tt.wantTok)
			}
		})
	}
}
