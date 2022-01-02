package main


import (
	"fmt"
	"io"
	"os"
	"strings"
)

type TokenType int

const (
	TokInvalid TokenType = iota
	TokNumber
	TokIdentifier
	TokString
	TokLparen
	TokRparen
	TokDot
	TokQuote
)

type Token struct {
	Offset int
	Length int
	Type   TokenType
}

type TokensFormatter struct {
	Source string
	Tokens []Token
}

type LexState int

const (
	LexIdle LexState = iota
	LexNumber
	LexIdentifier
	LexComment
	LexString
	LexStringEscaped
)

type Lex struct {
	Source strings.Builder
	Tokens []Token
	State  LexState
}

type AtomType int

const (
	AtomInvalid AtomType = iota
	AtomNumber
	AtomIdentifier
	AtomString
)

type Atom struct {
	Representation string
	Type           AtomType
}

type Expression struct {
	Atom  Atom
	Left  *Expression
	Right *Expression
}

type Pars struct {
	Lex    Lex
	Tree   []Expression
	tokens []Token
}

type Error struct {
	LineNumber   int
	OffsetInLine int
	Text         string
}

func (e Error) Error() string {
	return fmt.Sprintf("%v:%v: %v", e.LineNumber, e.OffsetInLine, e.Text)
}

func NewError(source string, offset int, text string) Error {
	var line int = 1
	var offsetInLine int
	var prev byte
	for i, c := range source {
		if i == offset {
			// The length of source may exceed greatly the offset of error location.
			// That's why we break here when the desired offset has been reached.
			break
		}
		offsetInLine += 1
		if prev == '\r' && c == '\n' {
			// Skip, because line was already incremented while we
			// were parsing single '\r' (see next else-if branch).
			offsetInLine = 0
		} else if c == '\r' || c == '\n' {
			line += 1
			offsetInLine = 0
		}
	}
	return Error{line, offsetInLine+1, text}
}

func (token Token) String() string {
	switch token.Type {
	case TokInvalid:
		return "TokInvalid"
	case TokNumber:
		return "TokNumber"
	case TokIdentifier:
		return "TokIdentifier"
	case TokString:
		return "TokString"
	case TokLparen:
		return "TokLparen"
	case TokRparen:
		return "TokRparen"
	case TokDot:
		return "TokDot"
	case TokQuote:
		return "TokQuote"
	}
	panic(fmt.Sprintf("Unknown token type %v", token.Type))
}

func (tfmt TokensFormatter) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range tfmt.Tokens {
		if i != 0 {
			sb.WriteString(", ")
		}
		literal := tfmt.Source[v.Offset : v.Offset+v.Length]
		sb.WriteString(fmt.Sprintf("%v<%v>", v, literal))
	}
	sb.WriteString("]")
	return sb.String()
}

func (lex Lex) String() string {
	return TokensFormatter{lex.Source.String(), lex.Tokens}.String()
}

func (atom Atom) String() string {
	switch atom.Type {
	case AtomInvalid:
		return fmt.Sprintf("AtomInvalid<%v>", atom.Representation)
	case AtomNumber:
		return fmt.Sprintf("%v", atom.Representation)
	case AtomIdentifier:
		return fmt.Sprintf("%v", atom.Representation)
	case AtomString:
		return fmt.Sprintf("%v", atom.Representation)
	}
	panic(fmt.Sprintf("Unknown atom type %v", atom.Type))
}

func (expression Expression) String() string {
	var sb strings.Builder
	if expression.Atom.Type != AtomInvalid {
		sb.WriteString(expression.Atom.String())
	} else {
		sb.WriteString("(")
		if expression.Left != nil || expression.Right != nil {
			if expression.Left == nil {
				sb.WriteString("()")
			} else {
				sb.WriteString(expression.Left.String())
			}
			sb.WriteString(" . ")
			if expression.Right == nil {
				sb.WriteString("()")
			} else {
				sb.WriteString(expression.Right.String())
			}
		}
		sb.WriteString(")")
	}
	return sb.String()
}

func (self *Lex) AddToken(t TokenType) []Token {
	var newTokens []Token
	if self.State != LexIdle {
		self.State = LexIdle
		newTokens = append(newTokens, self.Tokens[len(self.Tokens)-1])
	}
	newToken := Token{self.Source.Len(), 1, t}
	newTokens = append(newTokens, newToken)
	self.Tokens = append(self.Tokens, newToken)
	return newTokens
}

func (self *Lex) BeginNumber() {
	newToken := Token{self.Source.Len(), 1, TokNumber}
	self.Tokens = append(self.Tokens, newToken)
	self.State = LexNumber
}

func (self *Lex) BeginIdentifier() {
	newToken := Token{self.Source.Len(), 1, TokIdentifier}
	self.Tokens = append(self.Tokens, newToken)
	self.State = LexIdentifier
}

func (self *Lex) BeginString() []Token {
	var newTokens []Token
	if self.State != LexIdle {
		newTokens = append(newTokens, self.Tokens[len(self.Tokens)-1])
	}
	newToken := Token{self.Source.Len(), 1, TokString}
	self.Tokens = append(self.Tokens, newToken)
	self.State = LexString
	return newTokens
}

func (self *Lex) FinishBuiltin(t TokenType) []Token {
	token := self.Tokens[len(self.Tokens)-1]
	token.Length += 1
	token.Type = t
	self.Tokens[len(self.Tokens)-1] = token
	self.State = LexIdle
	return []Token{token}
}

func IsNumeric(c byte) bool {
	return c >= '0' && c <= '9'
}

func IsAlphabetic(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == 'w' || c == 'x' ||
		c == 'y' || c == 'z' || c == '-' || c == '!' || c == '$' || c == '%' || c == '*' ||
		c == '+' || c == '?' || c == '&' || c == '.' || c == '\\' || c == '/' || c == '~' ||
		c == '`' || c == ':' || c == '=' || c == '<' || c == '>' ||
		c == '^' || c == '#'
}

func IsCommentCharacter(c byte) bool {
	return c == '\t' || (c >= ' ' && c <= '~')
}

func IsStringCharacter(c byte) bool {
	return c == '\t' || c == '\n' || c == '\r' || (c >= ' ' && c <= '~')
}

func IsAlphaNumeric(c byte) bool {
	return IsNumeric(c) || IsAlphabetic(c) || c == '!' || c == '$'
}

func IsSingleCharToken(c byte) bool {
	return c == '(' || c == ')' || c == '.' || c == '\''
}

func TokenFromByte(c byte) TokenType {
	if c == '(' {
		return TokLparen
	} else if c == ')' {
		return TokRparen
	} else if c == '.' {
		return TokDot
	} else if c == '\'' {
		return TokQuote
	}
	panic(fmt.Sprintf("Byte %v cannot be converted to token", c))
}

func (self *Lex) ConsumeImpl(c byte) ([]Token, error) {
	switch self.State {
	case LexIdle:
		if IsSingleCharToken(c) {
			return self.AddToken(TokenFromByte(c)), nil
		} else if c == ' ' || c == 0x0A || c == 0x0D || c == '\t' {
			// Skip
		} else if IsNumeric(c) {
			self.BeginNumber()
		} else if IsAlphaNumeric(c) {
			self.BeginIdentifier()
		} else if c == '"' {
			return self.BeginString(), nil
		} else if c == ';' {
			self.State = LexComment
		} else {
			return []Token{}, NewError(
				self.Source.String(),
				self.Source.Len(),
				fmt.Sprintf("unexpected byte '%v'", c))
		}
	case LexNumber:
		if IsSingleCharToken(c) {
			return self.AddToken(TokenFromByte(c)), nil
		} else if c == '"' {
			self.BeginString()
		} else if c == ' ' || c == 0x0A || c == 0x0D || c == '\t' {
			if self.State != LexIdle {
				self.State = LexIdle
				return self.Tokens[len(self.Tokens)-1:], nil
			}
		} else if IsNumeric(c) {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
		} else if IsAlphaNumeric(c) {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			token.Type = TokIdentifier
			self.Tokens[len(self.Tokens)-1] = token
		} else if c == ';' {
			self.State = LexComment
		} else {
			return []Token{}, NewError(
				self.Source.String(),
				self.Source.Len(),
				fmt.Sprintf("unexpected byte '%v'", c))
		}
	case LexIdentifier:
		if IsSingleCharToken(c) {
			return self.AddToken(TokenFromByte(c)), nil
		} else if c == '"' {
			self.BeginString()
		} else if c == ' ' || c == 0x0A || c == 0x0D || c == '\t' {
			if self.State != LexIdle {
				self.State = LexIdle
				return self.Tokens[len(self.Tokens)-1:], nil
			}
		} else if IsAlphaNumeric(c) {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
		} else if c == ';' {
			self.State = LexComment
		} else {
			return []Token{}, NewError(
				self.Source.String(),
				self.Source.Len(),
				fmt.Sprintf("unexpected byte '%v'", c))
		}
	case LexComment:
		if c == 0x0A || c == 0x0D {
			self.State = LexIdle
		} else if IsCommentCharacter(c) {
			// Skip
		} else {
			return []Token{}, NewError(
				self.Source.String(),
				self.Source.Len(),
				fmt.Sprintf("unexpected byte '%v'", c))
		}
	case LexStringEscaped:
		if c == '\\' {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
		} else if IsStringCharacter(c) {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
		} else {
			return []Token{}, NewError(
				self.Source.String(),
				self.Source.Len(),
				fmt.Sprintf("unexpected byte '%v'", c))
		}
		self.State = LexString
	case LexString:
		if c == '"' {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
			self.State = LexIdle
			return self.Tokens[len(self.Tokens)-1:], nil
		} else if c == '\\' {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
			self.State = LexStringEscaped
		} else if IsStringCharacter(c) {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
		} else {
			return []Token{}, NewError(
				self.Source.String(),
				self.Source.Len(),
				fmt.Sprintf("unexpected byte '%v'", c))
		}
	}
	return []Token{}, nil
}

func (self *Lex) Consume(c byte) ([]Token, error) {
	newTokens, err := self.ConsumeImpl(c)
	self.Source.WriteByte(c)
	return newTokens, err
}

func AtomTypeFromToken(t TokenType) AtomType {
	switch t {
	case TokNumber:
		return AtomNumber
	case TokIdentifier:
		return AtomIdentifier
	case TokString:
		return AtomString
	}
	panic(fmt.Sprintf("Cannot convert TokenType %v to AtomType", t))
}

func NewAtomExpression(lex Lex, token Token) Expression {
	start, end := token.Offset, token.Offset+token.Length
	atom := Atom{lex.Source.String()[start:end], AtomTypeFromToken(token.Type)}
	return Expression{atom, nil, nil}
}

func NilExpression() Expression {
	return Expression{Atom{"", AtomInvalid}, nil, nil}
}

func (self *Pars) NextToken(input io.Reader) (t Token, err error) {
	for {
		if len(self.tokens) > 0 {
			token := self.tokens[0]
			self.tokens = self.tokens[1:]
			return token, nil
		}
		var c []byte = []byte{0}
		_, err := input.Read(c)
		if err != nil {
			return Token{0, 0, TokInvalid}, err
		}
		newTokens, err := self.Lex.Consume(c[0])
		if err != nil {
			return Token{0, 0, TokInvalid}, err
		}
		self.tokens = append(self.tokens, newTokens...)
	}
}

func (self *Pars) ParseNextExpression(input io.Reader, parentToken Token) (e Expression, err error) {
	if parentToken.Type == TokRparen {
		// Safety measure
		panic(fmt.Sprintf("Delegated unexpected %v", parentToken))
	}
	var token = parentToken
	if parentToken.Type == TokInvalid {
		localToken, err := self.NextToken(input)
		if err != nil {
			return NilExpression(), err
		}
		token = localToken
	}
	if token.Type == TokIdentifier || token.Type == TokNumber || token.Type == TokString {
		return NewAtomExpression(self.Lex, token), nil
	} else if token.Type == TokLparen {
		var rootExpression Expression
		token, err := self.NextToken(input)
		if err != nil {
			return NilExpression(), err
		}
		if token.Type != TokRparen {
			left, err := self.ParseNextExpression(input, token)
			if err != nil {
				return NilExpression(), err
			}
			rootExpression.Left = &left
			var lastExpression *Expression = &rootExpression
			for {
				var expression Expression
				lastExpression.Right = &expression
				token, err := self.NextToken(input)
				if err != nil {
					return NilExpression(), err
				}
				if token.Type == TokDot {
					right, err := self.ParseNextExpression(input, Token{0, 0, TokInvalid})
					if err != nil {
						return NilExpression(), err
					}
					expression = right
					token, err := self.NextToken(input)
					if err != nil {
						return NilExpression(), err
					}
					if token.Type != TokRparen {
						return NilExpression(), NewError(
							self.Lex.Source.String(),
							token.Offset,
							fmt.Sprintf(
								"Unexpected token %v, expected TokRparen<)>",
								TokensFormatter{self.Lex.Source.String(), []Token{token}}.String()))
					}
					break
				} else {
					if token.Type == TokRparen {
						break
					}
					left, err := self.ParseNextExpression(input, token)
					if err != nil {
						return NilExpression(), err
					}
					expression.Left = &left
					lastExpression = &expression
				}
			}
		}
		return rootExpression, nil
	} else if token.Type == TokQuote {
		expression := Expression{
			Atom{"", AtomInvalid},
			&Expression{
				Atom{"quote", AtomIdentifier},
				&Expression{
					Atom{"", AtomInvalid},
					nil,
					nil,
				},
				nil,
			},
			nil,
		}
		right, err := self.ParseNextExpression(input, Token{0, 0, TokInvalid})
		if err != nil {
			return NilExpression(), err
		}
		expression.Right = &Expression{
			Atom{"", AtomInvalid},
			&right,
			nil,
		}
		return expression, nil
	} else {
		return NilExpression(), NewError(
			self.Lex.Source.String(),
			token.Offset,
			fmt.Sprintf("Unsupported token %v", token))
	}
}

func TestLex() {
	var lex Lex
	var tokensNum int
	var tokens []Token
	for {
		var c []byte = []byte{0}
		_, err := os.Stdin.Read(c)
		if err != nil {
			break
		}
		newTokens, err := lex.Consume(c[0])
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		tokens = append(tokens, newTokens...)
		if c[0] == '\n' && len(lex.Tokens) > tokensNum {
			fmt.Println(TokensFormatter{lex.Source.String(), tokens})
			tokens = tokens[:0]
		}
	}
}

func main() {
	var parser Pars
	for {
		expression, err := parser.ParseNextExpression(os.Stdin, Token{0, 0, TokInvalid})
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		fmt.Printf("Expression: %v\n", expression)
	}
}
