package jsonpath

import (
	"bufio"
	"errors"
	"unicode"
)

type lexer struct {
	r   *bufio.Reader
	err error
	out interface{}
}

func (l *lexer) Lex(lval *yySymType) int {
	for {
		r, _, err := l.r.ReadRune()
		if err != nil {
			return -1
		}

		switch {
		case unicode.IsSpace(r):
			continue

		case unicode.IsLetter(r) || unicode.IsNumber(r):
			rs := []rune{r}
			for {
				r, _, err = l.r.ReadRune()
				if err != nil {
					break
				}

				if !(unicode.IsLetter(r) || unicode.IsNumber(r)) {
					l.r.UnreadRune()
					break
				}

				rs = append(rs, r)
			}

			lval.s = string(rs)
			return STRING

		case r == '\'':
			rs := []rune{}
			for {
				r, _, err = l.r.ReadRune()
				if err != nil {
					return -1
				}

				if r == '\'' && (len(rs) == 0 || rs[len(rs)-1] != '\\') {
					break
				}

				rs = append(rs, r)
			}

			lval.s = string(rs)
			return STRING

		default:
			return int(r)
		}
	}
}

func (l *lexer) Error(s string) {
	l.err = errors.New(s)
}
