package main

import (
	"fmt"
	"os"
	"strings"
)

type TokenType int

const (
	TokNumber TokenType = iota
	TokBuiltinDefine
	TokBuiltinLambda
	TokBuiltinQuote
	TokBuiltinCar
	TokBuiltinCdr
	TokIdentifier
	TokString
	TokLparen
	TokRparen
	TokDot
)

type Token struct {
	Offset int
	Length int
	Type   TokenType
}

func (token Token) String() string {
	switch token.Type {
	case TokNumber:
		return "TokNumber"
	case TokBuiltinDefine:
		return "TokBuiltinDefine"
	case TokBuiltinLambda:
		return "TokBuiltinLambda"
	case TokBuiltinQuote:
		return "TokBuiltinQuote"
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
	case TokBuiltinCar:
		return "TokBuiltinCar"
	case TokBuiltinCdr:
		return "TokBuiltinCdr"
	}
	return "<?>"
}

type State int

const (
	LexIdle State = iota
	LexNumber
	LexIdentifier
	LexComment
	LexString
	LexStringEscaped
)

type TokensFormatter struct {
	Source string
	Tokens []Token
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

type Lex struct {
	// TODO use dynamically allocated huge freaking buffer to optimize tokenization of builtins
	Source strings.Builder
	Tokens []Token
	State  State
}

func (lex Lex) String() string {
	return TokensFormatter{lex.Source.String(), lex.Tokens}.String()
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
	panic("")
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

func IsCharacter(c byte) bool {
	return c == 0x09 || (c >= ' ' && c <= '~')
}

func IsAlphaNumeric(c byte) bool {
	return IsNumeric(c) || IsAlphabetic(c) || c == '!' || c == '$'
}

func IsSingleCharToken(c byte) bool {
	return c == '(' || c == ')' || c == '.'
}

func TokenFromByte(c byte) TokenType {
	if c == '(' {
		return TokLparen
	} else if c == ')' {
		return TokRparen
	} else if c == '.' {
		return TokDot
	}
	panic(fmt.Sprintf("Byte %v cannot be converted to token", c))
	return TokDot
}

func (self Lex) IsBuiltinComposition(c byte) bool {
	var builtins []string = []string{
		"car",
		"cdr",
		"quote",
		"define",
		"lambda",
	}
	//var largestLength = len(builtins[len(builtins)-1])
	for _, builtin := range(builtins) {
		if self.Source.String()[self.Source.Len()-self.Tokens[len(self.Tokens)-1].Length:] + string([]byte{c}) == builtin {
			return true
		}
	}
	return false
}

func (self Lex) BuiltinWithByte(c byte) TokenType {
	literal := self.Source.String()[self.Source.Len()-self.Tokens[len(self.Tokens)-1].Length:] + string([]byte{c})
	switch literal {
	case "car":
		return TokBuiltinCar
	case "cdr":
		return TokBuiltinCdr
	case "quote":
		return TokBuiltinQuote
	case "define":
		return TokBuiltinDefine
	case "lambda":
		return TokBuiltinLambda
	}
	panic(fmt.Sprintf("No known conversion to TokenType for \"%v\"", literal))
	return TokBuiltinLambda
}

func (self *Lex) ConsumeImpl(c byte) []Token {
	// TODO support quote syntax like "(car '(1 2 3))"
	switch self.State {
	case LexIdle:
		if IsSingleCharToken(c) {
			return self.AddToken(TokenFromByte(c))
		} else if c == ' ' || c == 0x0A || c == 0x0D {
			// Skip
		} else if IsNumeric(c) {
			self.BeginNumber()
		} else if IsAlphaNumeric(c) {
			self.BeginIdentifier()
		} else if c == '"' {
			return self.BeginString()
		} else if c == ';' {
			self.State = LexComment
		} else {
			// TODO raise error
			panic(fmt.Sprintf("unexpected byte '%v'", c))
		}
	case LexNumber:
		if IsSingleCharToken(c) {
			return self.AddToken(TokenFromByte(c))
		} else if c == '"' {
			self.BeginString()
		} else if c == ' ' || c == 0x0A || c == 0x0D {
			if self.State != LexIdle {
				self.State = LexIdle
				return self.Tokens[len(self.Tokens)-1:]
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
			// TODO raise error
			panic(fmt.Sprintf("unexpected byte '%v'", c))
		}
	case LexIdentifier:
		if self.IsBuiltinComposition(c) {
			return self.FinishBuiltin(self.BuiltinWithByte(c))
		} else if IsSingleCharToken(c) {
			return self.AddToken(TokenFromByte(c))
		} else if c == '"' {
			self.BeginString()
		} else if c == ' ' || c == 0x0A || c == 0x0D {
			if self.State != LexIdle {
				self.State = LexIdle
				return self.Tokens[len(self.Tokens)-1:]
			}
		} else if IsAlphaNumeric(c) {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
		} else if c == ';' {
			self.State = LexComment
		} else {
			// TODO raise error
			panic(fmt.Sprintf("unexpected byte '%v'", c))
		}
	case LexComment:
		if c == 0x0A {
			self.State = LexIdle
		} else if IsCharacter(c) {
			// Skip
		} else {
			// TODO raise error
			panic(fmt.Sprintf("unexpected byte '%v'", c))
		}
	case LexStringEscaped:
		if c == '\\' {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
		} else if IsCharacter(c) {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
		} else {
			// TODO raise error
			panic(fmt.Sprintf("unexpected byte '%v'", c))
		}
		self.State = LexString
	case LexString:
		if c == '"' {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
			self.State = LexIdle
			return self.Tokens[len(self.Tokens)-1:]
		} else if c == '\\' {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
			self.State = LexStringEscaped
		} else if IsCharacter(c) {
			token := self.Tokens[len(self.Tokens)-1]
			token.Length += 1
			self.Tokens[len(self.Tokens)-1] = token
		} else {
			// TODO raise error
			panic(fmt.Sprintf("unexpected byte '%v'", c))
		}
	}
	return []Token{}
}

func (self *Lex) Consume(c byte) []Token {
	newTokens := self.ConsumeImpl(c)
	self.Source.WriteByte(c)
	return newTokens
}

func main() {
	var lex Lex
	var tokensNum int
	var tokens []Token
	for {
		var c []byte = []byte{0}
		_, err := os.Stdin.Read(c)
		if err != nil {
			break
		}
		tokens = append(tokens, lex.Consume(c[0])...)
		if c[0] == '\n' && len(lex.Tokens) > tokensNum {
			fmt.Println(TokensFormatter{lex.Source.String(), tokens})
			tokens = tokens[:0]
		}
	}
}
