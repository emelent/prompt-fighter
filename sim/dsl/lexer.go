package dsl

import (
	"fmt"
	"sort"
	"strconv"
)

// Pos is a 1-based source position; Col counts bytes.
type Pos struct {
	Line, Col int
}

func (p Pos) before(q Pos) bool {
	return p.Line < q.Line || (p.Line == q.Line && p.Col < q.Col)
}

// Error is a validation error at a source position.
type Error struct {
	Line, Col int
	Msg       string
}

func (e Error) String() string { return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, e.Msg) }
func (e Error) Error() string  { return e.String() }

func errAt(p Pos, format string, args ...any) Error {
	return Error{Line: p.Line, Col: p.Col, Msg: fmt.Sprintf(format, args...)}
}

func sortErrors(errs []Error, max int) []Error {
	sort.SliceStable(errs, func(i, j int) bool {
		return Pos{errs[i].Line, errs[i].Col}.before(Pos{errs[j].Line, errs[j].Col})
	})
	if len(errs) > max {
		errs = errs[:max]
	}
	return errs
}

// TokKind classifies tokens.
type TokKind int

const (
	TEOF TokKind = iota
	TIdent
	TInt
	TCount // xN
	TLBrace
	TRBrace
	TLParen
	TRParen
	TCmp
)

// Token is a lexical token.
type Token struct {
	Kind TokKind
	Text string
	Val  int
	Pos  Pos
}

const intCap = 1_000_000_000

func isLetter(c byte) bool { return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isDigit(c byte) bool  { return c >= '0' && c <= '9' }

func atoiCapped(s string) int {
	if len(s) > 9 {
		return intCap
	}
	v, _ := strconv.Atoi(s)
	return v
}

// Lex tokenizes src. Common foreign operators (&&, ||, !, =, ;) produce an
// error with a hint and are replaced by their .pf equivalent so parsing can
// continue.
func Lex(src string) ([]Token, []Error) {
	var toks []Token
	var errs []Error
	line, col := 1, 1
	i := 0
	adv := func(n int) {
		for k := 0; k < n; k++ {
			if src[i] == '\n' {
				line++
				col = 1
			} else {
				col++
			}
			i++
		}
	}
	for i < len(src) {
		c := src[i]
		p := Pos{line, col}
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			adv(1)
		case c == '#':
			for i < len(src) && src[i] != '\n' {
				adv(1)
			}
		case isLetter(c):
			j := i
			for j < len(src) && (isLetter(src[j]) || isDigit(src[j])) {
				j++
			}
			if j < len(src) && src[j] == '.' && j+1 < len(src) && isLetter(src[j+1]) {
				j++
				for j < len(src) && (isLetter(src[j]) || isDigit(src[j])) {
					j++
				}
			}
			word := src[i:j]
			if len(word) > 1 && word[0] == 'x' && allDigits(word[1:]) {
				toks = append(toks, Token{Kind: TCount, Text: word, Val: atoiCapped(word[1:]), Pos: p})
			} else {
				toks = append(toks, Token{Kind: TIdent, Text: word, Pos: p})
			}
			adv(j - i)
		case isDigit(c):
			j := i
			for j < len(src) && isDigit(src[j]) {
				j++
			}
			toks = append(toks, Token{Kind: TInt, Text: src[i:j], Val: atoiCapped(src[i:j]), Pos: p})
			adv(j - i)
		case c == '{':
			toks = append(toks, Token{Kind: TLBrace, Text: "{", Pos: p})
			adv(1)
		case c == '}':
			toks = append(toks, Token{Kind: TRBrace, Text: "}", Pos: p})
			adv(1)
		case c == '(':
			toks = append(toks, Token{Kind: TLParen, Text: "(", Pos: p})
			adv(1)
		case c == ')':
			toks = append(toks, Token{Kind: TRParen, Text: ")", Pos: p})
			adv(1)
		case c == '<' || c == '>':
			if i+1 < len(src) && src[i+1] == '=' {
				toks = append(toks, Token{Kind: TCmp, Text: src[i : i+2], Pos: p})
				adv(2)
			} else {
				toks = append(toks, Token{Kind: TCmp, Text: src[i : i+1], Pos: p})
				adv(1)
			}
		case c == '=':
			if i+1 < len(src) && src[i+1] == '=' {
				toks = append(toks, Token{Kind: TCmp, Text: "==", Pos: p})
				adv(2)
			} else {
				errs = append(errs, errAt(p, "use `==` to compare, not `=`"))
				toks = append(toks, Token{Kind: TCmp, Text: "==", Pos: p})
				adv(1)
			}
		case c == '!':
			if i+1 < len(src) && src[i+1] == '=' {
				toks = append(toks, Token{Kind: TCmp, Text: "!=", Pos: p})
				adv(2)
			} else {
				errs = append(errs, errAt(p, "use `not` instead of `!`"))
				toks = append(toks, Token{Kind: TIdent, Text: "not", Pos: p})
				adv(1)
			}
		case c == '&' || c == '|':
			word, n := "and", 1
			if c == '|' {
				word = "or"
			}
			if i+1 < len(src) && src[i+1] == c {
				n = 2
			}
			errs = append(errs, errAt(p, "use `%s` instead of `%s`", word, src[i:i+n]))
			toks = append(toks, Token{Kind: TIdent, Text: word, Pos: p})
			adv(n)
		case c == ';':
			errs = append(errs, errAt(p, "`;` is not needed; separate statements with spaces or newlines"))
			adv(1)
		default:
			errs = append(errs, errAt(p, "unexpected character %q", rune(c)))
			adv(1)
		}
	}
	toks = append(toks, Token{Kind: TEOF, Text: "end of script", Pos: Pos{line, col}})
	return toks, errs
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return len(s) > 0
}
