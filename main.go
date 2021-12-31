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
	Type TokenType
}

func (token Token) String() string {
	switch token.Type {
	case TokNumber: return "TokNumber"
	case TokBuiltinDefun: return "TokBuiltinDefun"
	case TokBuiltinLambda: return "TokBuiltinLambda"
	case TokBuiltinSetq: return "TokBuiltinSetq"
	case TokIdentifier: return "TokIdentifier"
	case TokString: return "TokString"
	case TokLparen: return "TokLparen"
	case TokRparen: return "TokRparen"
	case TokDot: return "TokDot"
	case TokCar: return "TokCar"
	case TokCdr: return "TokCdr"
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
	Source strings.Builder
	Tokens []Token
	State State
	TokenBeginOffset int
}

func (lex Lex) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range(lex.Tokens) {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%v<%v>", v, lex.Source.String()[v.Offset:v.Offset+v.Length]))
	}
	sb.WriteString("]")
	return sb.String()
}

func (self *Lex) AddToken(t TokenType) {
	self.Tokens = append(self.Tokens, Token{self.Source.Len(), 1, t});
}

func IsNumeric(c byte) bool {
	return c >= '0' && c <= '9';
}

func IsAlphabetic(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c == '_');

}

func IsAlphaNumeric(c byte) bool {
	return IsNumeric(c) || IsAlphabetic(c)
}

func (self *Lex) Consume(c byte) {
	switch c {
	case '(': self.AddToken(TokLparen)
	case ')': self.AddToken(TokRparen)
	case '.': self.AddToken(TokDot)
	case IsAlphabetic(c):
		self.TokenBeginOffset = self.Source.Len()
		self.State = LexIdentifier
	case IsNumeric(c):
		self.TokenBeginOffset = self.Source.Len()
		self.State = LexNumber
	case ' ', 0x0A, 0x0D: // Skip
	default:
		// TODO raise error
		panic("unexpected byte")
	}
	self.Source.WriteByte(c)
}

func main() {
	var lex Lex;
	for {
		var c []byte = []byte{0}
		_, err := os.Stdin.Read(c)
		if err != nil {
			break
		}
		lex.Consume(c[0])
	}
	fmt.Println(lex);
}
