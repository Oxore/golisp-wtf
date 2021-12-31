package main

import (
	"fmt"
	"os"
	"strings"
)

type TokenType int

const (
	TokNumber TokenType = iota
	TokBuiltinDefun
	TokBuiltinLambda
	TokBuiltinSetq
	TokIdentifier
	TokString
	TokLparen
	TokRparen
	TokDot
	TokCar
	TokCdr
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
	case TokBuiltinDefun:
		return "TokBuiltinDefun"
	case TokBuiltinLambda:
		return "TokBuiltinLambda"
	case TokBuiltinSetq:
		return "TokBuiltinSetq"
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
	case TokCar:
		return "TokCar"
	case TokCdr:
		return "TokCdr"
	}
	return "<?>"
}

type State int

const (
	LexIdle State = iota
	LexNumber
	LexIdentifier
)

type Lex struct {
	Source           strings.Builder
	Tokens           []Token
	State            State
}

func (lex Lex) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range lex.Tokens {
		if i != 0 {
			sb.WriteString(", ")
		}
		literal := lex.Source.String()[v.Offset : v.Offset+v.Length]
		sb.WriteString(fmt.Sprintf("%v<%v>", v, literal))
	}
	sb.WriteString("]")
	return sb.String()
}

func (self *Lex) AddToken(t TokenType) {
	self.State = LexIdle
	self.Tokens = append(self.Tokens, Token{self.Source.Len(), 1, t})
}

func (self *Lex) BeginNumber() {
	self.Tokens = append(self.Tokens, Token{self.Source.Len(), 1, TokNumber})
	self.State = LexNumber
}

func (self *Lex) BeginIdentifier() {
	self.Tokens = append(self.Tokens, Token{self.Source.Len(), 1, TokIdentifier})
	self.State = LexIdentifier
}

func IsNumeric(c byte) bool {
	return c >= '0' && c <= '9'
}

func IsAlphabetic(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c == '_')
}

func IsAlphaNumeric(c byte) bool {
	return IsNumeric(c) || IsAlphabetic(c)
}

func (self *Lex) Consume(c byte) {
	if c == '(' {
		self.AddToken(TokLparen)
	} else if c == ')' {
		self.AddToken(TokRparen)
	} else if c == '.' {
		self.AddToken(TokDot)
	} else if c == ' ' || c == 0x0A || c == 0x0D {
		if self.State != LexIdle {
			self.State = LexIdle
		}
	} else {
		switch self.State {
		case LexIdle:
			if IsNumeric(c) {
				self.BeginNumber()
			} else if IsAlphaNumeric(c) {
				self.BeginIdentifier()
			} else {
				// TODO raise error
				panic("unexpected byte")
			}
		case LexNumber:
			if IsNumeric(c) {
				token := self.Tokens[len(self.Tokens)-1]
				token.Length += 1
				self.Tokens[len(self.Tokens)-1] = token
			} else if IsAlphaNumeric(c) {
				token := self.Tokens[len(self.Tokens)-1]
				token.Length += 1
				token.Type = TokIdentifier
				self.Tokens[len(self.Tokens)-1] = token
			} else {
				// TODO raise error
				panic("unexpected byte")
			}
		case LexIdentifier:
			if IsAlphaNumeric(c) {
				token := self.Tokens[len(self.Tokens)-1]
				token.Length += 1
				self.Tokens[len(self.Tokens)-1] = token
			} else {
				// TODO raise error
				panic("unexpected byte")
			}
		}
	}
	self.Source.WriteByte(c)
}

func main() {
	var lex Lex
	var tokensNum int
	for {
		var c []byte = []byte{0}
		_, err := os.Stdin.Read(c)
		if err != nil {
			break
		}
		lex.Consume(c[0])
		if c[0] == '\n' && len(lex.Tokens) > tokensNum {
			lexTmp := lex
			lexTmp.Tokens = lex.Tokens[tokensNum:]
			fmt.Println(lexTmp)
			tokensNum = len(lex.Tokens)
		}
	}
}
