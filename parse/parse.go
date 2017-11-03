package parse

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/faiface/lambda/ast"
)

type MetaInfo struct {
	*FileInfo
	Name string
}

type FileInfo struct {
	Filename     string
	Line, Column int
}

type Error struct {
	*FileInfo
	Msg string
}

func (err *Error) Error() string {
	if err.FileInfo == nil {
		return err.Msg
	}
	return fmt.Sprintf("%s:%d:%d: %s", err.Filename, err.Line, err.Column, err.Msg)
}

func Single(filename string, r io.Reader) (ast.Node, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	toks := Tokenize(filename, data)
	if err != nil {
		return nil, err
	}
	return SingleFromTokens(toks)
}

func Definitions(filename string, r io.Reader) (map[string]ast.Node, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	toks := Tokenize(filename, data)
	if err != nil {
		return nil, err
	}
	return DefinitionsFromTokens(toks)
}

func SingleFromTokens(toks []Token) (ast.Node, error) {
	var node ast.Node

	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		var right ast.Node

		switch tok.Token {
		case "(":
			match := matchingParen(toks[i+1:])
			if match == -1 {
				return nil, &Error{
					FileInfo: tok.FileInfo,
					Msg:      "no matching right parenthesis",
				}
			}
			var err error
			right, err = SingleFromTokens(toks[i+1 : i+1+match])
			if err != nil {
				return nil, err
			}
			if right == nil {
				return nil, &Error{
					FileInfo: tok.FileInfo,
					Msg:      "no expression inside parenthesis",
				}
			}
			i += 1 + match
		case ")":
			return nil, &Error{
				FileInfo: tok.FileInfo,
				Msg:      "no matching left parenthesis",
			}
		case "\\", "λ":
			if len(toks[i+1:]) == 0 {
				return nil, &Error{
					FileInfo: tok.FileInfo,
					Msg:      fmt.Sprintf("no binding after '%s'", tok.Token),
				}
			}
			bound := toks[i+1].Token
			if bound == strings.Title(bound) {
				return nil, &Error{
					FileInfo: tok.FileInfo,
					Msg:      fmt.Sprintf("invalid bound name '%s'", bound),
				}
			}
			switch bound {
			case "(", ")", "\\", "λ", ";":
				return nil, &Error{
					FileInfo: tok.FileInfo,
					Msg:      fmt.Sprintf("invalid bound name '%s'", bound),
				}
			}
			body, err := SingleFromTokens(toks[i+2:])
			if err != nil {
				return nil, err
			}
			if body == nil {
				return nil, &Error{
					FileInfo: tok.FileInfo,
					Msg:      "no body in abstraction",
				}
			}
			return wrapAppl(node, &ast.Abst{
				Bound: toks[i+1].Token,
				Body:  body,
				Meta: &MetaInfo{
					FileInfo: tok.FileInfo,
					Name:     toks[i+1].Token,
				},
			}), nil
		case ";":
			afterColon, err := SingleFromTokens(toks[i+1:])
			if err != nil {
				return nil, err
			}
			if afterColon == nil {
				return nil, &Error{
					FileInfo: tok.FileInfo,
					Msg:      "no expression after ';'",
				}
			}
			return wrapAppl(node, afterColon), nil
		default:
			identifier := tok.Token
			if identifier == strings.Title(identifier) {
				right = &ast.Global{
					Name: identifier,
					Meta: &MetaInfo{
						FileInfo: tok.FileInfo,
						Name:     identifier,
					},
				}
			} else {
				right = &ast.Var{
					Name: identifier,
					Meta: &MetaInfo{
						FileInfo: tok.FileInfo,
						Name:     identifier,
					},
				}
			}
		}

		node = wrapAppl(node, right)
	}

	return node, nil
}

func matchingParen(toks []Token) int {
	var (
		i    int
		tok  Token
		nest = 1
	)
	for i, tok = range toks {
		switch tok.Token {
		case "(":
			nest++
		case ")":
			nest--
		}
		if nest == 0 {
			break
		}
	}
	if nest != 0 {
		return -1
	}
	return i
}

func wrapAppl(left, right ast.Node) ast.Node {
	if left == nil {
		return right
	}
	return &ast.Appl{
		Left:  left,
		Right: right,
		Meta:  nil,
	}
}

func DefinitionsFromTokens(toks []Token) (map[string]ast.Node, error) {
	defs := make(map[string]ast.Node)
	for len(toks) > 0 {
		name, node, ends, err := definition(toks)
		if err != nil {
			return nil, err
		}
		defs[name] = node
		toks = toks[ends:]
	}
	return defs, nil
}

func definition(toks []Token) (name string, node ast.Node, ends int, err error) {
	if len(toks) < 3 || toks[1].Token != "=" {
		return "", nil, 0, &Error{
			Msg: "no or invalid definition",
		}
	}
	name = toks[0].Token
	if name != strings.Title(name) || name == "=" {
		return "", nil, 0, &Error{
			FileInfo: toks[0].FileInfo,
			Msg:      fmt.Sprintf("invalid global name '%s'", name),
		}
	}
	for i := 2; i < len(toks); i++ {
		if toks[i].Token == "=" {
			ends = i - 1
			break
		}
	}
	if ends == 0 {
		ends = len(toks)
	}
	node, err = SingleFromTokens(toks[2:ends])
	if err != nil {
		return "", nil, 0, err
	}
	return name, node, ends, nil
}

func Tokenize(filename string, data []byte) []Token {
	var (
		tokens []Token
		token  []rune
		i      = 0
		line   = 1
		column = 1
	)

	flushToken := func(line, column int) {
		if len(token) == 0 {
			return
		}
		tokens = append(tokens, Token{
			FileInfo: &FileInfo{
				Filename: filename,
				Line:     line,
				Column:   column,
			},
			Token: string(token),
		})
		token = token[:0]
	}

nextToken:
	for i < len(data) {
		firstColumn := column
		for i < len(data) {
			r, width := utf8.DecodeRune(data[i:])
			i += width
			column++
			if r == '\n' {
				line++
				column = 1
			}
			if unicode.IsSpace(r) {
				break
			}
			switch r {
			case '(', ')', '\\', 'λ', ';':
				flushToken(line, firstColumn)
				token = append(token[:0], r)
				flushToken(line, column-1)
				continue nextToken
			default:
				token = append(token, r)
			}
		}
		flushToken(line, firstColumn)
	}

	return tokens
}

type Token struct {
	*FileInfo
	Token string
}
