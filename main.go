package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"strconv"
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
	state  LexState
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
	Offset         int
	Type           AtomType
}

type Expression struct {
	Atom  Atom
	Left  *Expression
	Right *Expression
}

type Pars struct {
	Lex    Lex
	tokens []Token
}

type Error struct {
	LineNumber   int
	OffsetInLine int
	Text         string
}

type ValueType int

const (
	ValNull ValueType = iota
	ValBool
	ValPair
	ValSymbol
	ValNumber
	ValChar
	ValString
	ValProc
)

type Value struct {
	Type ValueType
	Bool bool
	PairLeft *Value
	PairRight *Value
	Symbol string
	Number int
	Char byte
	StringData string
	Proc func(Value, Interp) (Value, error)
}

type Interp struct {
	Source *strings.Builder
	Table map[string] Value
}

func (e Error) Error() string {
	return fmt.Sprintf("<stdin>:%v:%v: %v", e.LineNumber, e.OffsetInLine, e.Text)
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
			// Do not increment line number, because line was already incremented while
			// we were parsing single '\r' (see next else-if branch).
			offsetInLine = 0
		} else if c == '\r' || c == '\n' {
			line += 1
			offsetInLine = 0
		}
	}
	return Error{line, offsetInLine + 1, text}
}

func (t ValueType) String() string {
	switch t {
	case ValNull:
		return "ValNull"
	case ValBool:
		return "ValBool"
	case ValPair:
		return "ValPair"
	case ValSymbol:
		return "ValSymbol"
	case ValNumber:
		return "ValNumber"
	case ValChar:
		return "ValChar"
	case ValString:
		return "ValString"
	case ValProc:
		return "ValProc"
	}
	panic(fmt.Sprintf("Unknown Value type %d", t))
}

func (v Value) String() string {
	switch v.Type {
	case ValNull:
		return fmt.Sprintf("%v<()>", v.Type)
	case ValBool:
		return fmt.Sprintf("%v<%t>", v.Type, v.Bool)
	case ValPair:
		return fmt.Sprintf("%v<(%v . %v)>", v.Type, v.PairLeft, v.PairRight)
	case ValSymbol:
		return fmt.Sprintf("%v<%s>", v.Type, v.Symbol)
	case ValNumber:
		return fmt.Sprintf("%v<%d>", v.Type, v.Number)
	case ValChar:
		return fmt.Sprintf("%v<%c>", v.Type, v.Char)
	case ValString:
		return fmt.Sprintf("%v<%s>", v.Type, v.StringData)
	case ValProc:
		panic("String() for ValProc is not implemented")
	}
	panic(fmt.Sprintf("Unknown Value type %d", v.Type))
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
	case AtomNumber, AtomIdentifier, AtomString:
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
	if self.state != LexIdle {
		self.state = LexIdle
		newTokens = append(newTokens, self.LastToken())
	}
	newToken := Token{self.Source.Len(), 1, t}
	newTokens = append(newTokens, newToken)
	self.Tokens = append(self.Tokens, newToken)
	return newTokens
}

func (self *Lex) BeginNumber() {
	newToken := Token{self.Source.Len(), 1, TokNumber}
	self.Tokens = append(self.Tokens, newToken)
	self.state = LexNumber
}

func (self *Lex) BeginIdentifier() {
	newToken := Token{self.Source.Len(), 1, TokIdentifier}
	self.Tokens = append(self.Tokens, newToken)
	self.state = LexIdentifier
}

func (self *Lex) BeginString() []Token {
	var newTokens []Token
	if self.state != LexIdle {
		newTokens = append(newTokens, self.LastToken())
	}
	newToken := Token{self.Source.Len(), 1, TokString}
	self.Tokens = append(self.Tokens, newToken)
	self.state = LexString
	return newTokens
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

func IsPrintableCharacter(c byte) bool {
	return c >= ' ' && c <= '~'
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
	switch c {
	case '(':
		return TokLparen
	case ')':
		return TokRparen
	case '.':
		return TokDot
	case '\'':
		return TokQuote
	}
	panic(fmt.Sprintf("Byte %v cannot be converted to token", c))
}

func (self Lex) NewUnexpectedByteError(c byte) error {
	var text string
	if IsPrintableCharacter(c) {
		text = fmt.Sprintf("unexpected byte '%c'", c)
	} else {
		text = fmt.Sprintf("unexpected byte 0x%X", c)
	}
	return NewError(self.Source.String(), self.Source.Len(), text)
}

func (self *Lex) LastTokenMut() *Token {
	return &self.Tokens[len(self.Tokens)-1]
}

func (self Lex) LastToken() Token {
	return self.Tokens[len(self.Tokens)-1]
}

func (self *Lex) ConsumeImpl(c byte) ([]Token, error) {
	switch self.state {
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
			self.state = LexComment
		} else {
			return []Token{}, self.NewUnexpectedByteError(c)
		}
	case LexNumber:
		if IsSingleCharToken(c) {
			return self.AddToken(TokenFromByte(c)), nil
		} else if c == '"' {
			self.BeginString()
		} else if c == ' ' || c == 0x0A || c == 0x0D || c == '\t' {
			self.state = LexIdle
			return []Token{self.LastToken()}, nil
		} else if IsNumeric(c) {
			self.LastTokenMut().Length += 1
		} else if IsAlphaNumeric(c) {
			self.LastTokenMut().Length += 1
			self.LastTokenMut().Type = TokIdentifier
		} else if c == ';' {
			self.state = LexComment
			return []Token{self.LastToken()}, nil
		} else {
			self.state = LexIdle
			return []Token{self.LastToken()}, self.NewUnexpectedByteError(c)
		}
	case LexIdentifier:
		if IsSingleCharToken(c) {
			return self.AddToken(TokenFromByte(c)), nil
		} else if c == '"' {
			self.BeginString()
		} else if c == ' ' || c == 0x0A || c == 0x0D || c == '\t' {
			self.state = LexIdle
			return []Token{self.LastToken()}, nil
		} else if IsAlphaNumeric(c) {
			self.LastTokenMut().Length += 1
		} else if c == ';' {
			self.state = LexComment
			return []Token{self.LastToken()}, nil
		} else {
			self.state = LexIdle
			return []Token{self.LastToken()}, self.NewUnexpectedByteError(c)
		}
	case LexComment:
		if c == 0x0A || c == 0x0D {
			self.state = LexIdle
		} else if IsCommentCharacter(c) {
			// Skip
		} else {
			self.state = LexIdle
			return []Token{}, self.NewUnexpectedByteError(c)
		}
	case LexStringEscaped:
		if IsStringCharacter(c) {
			self.LastTokenMut().Length += 1
			self.state = LexString
		} else {
			self.state = LexIdle
			return []Token{self.LastToken()}, self.NewUnexpectedByteError(c)
		}
		self.state = LexString
	case LexString:
		if c == '"' {
			self.LastTokenMut().Length += 1
			self.state = LexIdle
			return []Token{self.LastToken()}, nil
		} else if c == '\\' {
			self.LastTokenMut().Length += 1
			self.state = LexStringEscaped
		} else if IsStringCharacter(c) {
			self.LastTokenMut().Length += 1
		} else {
			self.state = LexIdle
			return []Token{self.LastToken()}, self.NewUnexpectedByteError(c)
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

func AtomFromToken(lex Lex, token Token) Expression {
	start, end := token.Offset, token.Offset+token.Length
	atom := Atom{lex.Source.String()[start:end], token.Offset, AtomTypeFromToken(token.Type)}
	return Expression{atom, nil, nil}
}

func ExpressionNil() Expression {
	return Expression{Atom{}, nil, nil}
}

func NewNode(left, right *Expression) *Expression {
	return &Expression{Atom{}, left, right}
}

func NewAtom(repr string, offset int, t AtomType) *Expression {
	return &Expression{Atom{repr, offset, t}, nil, nil}
}

func (self *Pars) NextToken(input io.Reader) (Token, error) {
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

func (self Pars) NewUnexpectedTokenError(token Token) error {
	return NewError(
		self.Lex.Source.String(),
		token.Offset,
		fmt.Sprintf(
			"Unexpected token %v",
			TokensFormatter{self.Lex.Source.String(), []Token{token}}.String()))
}

func (self *Pars) ParseRemainingList(input io.Reader) (*Expression, error) {
	var pseudoRoot Expression
	var last *Expression = &pseudoRoot
	for {
		var expression Expression
		last.Right = &expression
		token, err := self.NextToken(input)
		if err != nil {
			return pseudoRoot.Right, err
		}
		if token.Type == TokDot {
			// A pretty much determined sequence is expected here after we got the
			// TokDot token type
			right, err := self.ParseNext(input)
			if err != nil {
				return pseudoRoot.Right, err
			}
			expression = right
			token, err := self.NextToken(input)
			if err != nil {
				return pseudoRoot.Right, err
			}
			if token.Type != TokRparen {
				return pseudoRoot.Right, self.NewUnexpectedTokenError(token)
			}
			return pseudoRoot.Right, nil
		}
		if token.Type == TokRparen {
			return pseudoRoot.Right, nil
		}
		left, err := self.ParseNextWithToken(input, token)
		if err != nil {
			return pseudoRoot.Right, err
		}
		expression.Left = &left
		last = &expression
	}
}

func (self *Pars) ParseNextWithToken(input io.Reader, parentToken Token) (Expression, error) {
	if parentToken.Type == TokRparen {
		// Safety measure
		panic(fmt.Sprintf("Delegated unexpected %v", parentToken))
	}
	token := parentToken
	if parentToken.Type == TokInvalid {
		newToken, err := self.NextToken(input)
		if err != nil {
			return ExpressionNil(), err
		}
		token = newToken
	}
	switch token.Type {
	case TokIdentifier, TokNumber, TokString:
		return AtomFromToken(self.Lex, token), nil
	case TokLparen:
		token, err := self.NextToken(input)
		if err != nil {
			return ExpressionNil(), err
		}
		if token.Type == TokRparen {
			return ExpressionNil(), nil
		}
		left, err := self.ParseNextWithToken(input, token)
		if err != nil {
			return ExpressionNil(), err
		}
		right, err := self.ParseRemainingList(input)
		return *NewNode(&left, right), err
	case TokQuote:
		quoted, err := self.ParseNext(input)
		return *NewNode(NewAtom("quote", token.Offset, AtomIdentifier), NewNode(&quoted, nil)), err
	}
	return ExpressionNil(), self.NewUnexpectedTokenError(token)
}

func (self *Pars) ParseNext(input io.Reader) (Expression, error) {
	return self.ParseNextWithToken(input, Token{0, 0, TokInvalid})
}

func IsNilExpression(e Expression) bool {
	return e.Atom.Type == AtomInvalid && e.Right == nil && e.Left == nil
}

func (self *Interp) EvalRight(expression Expression) (Value, error) {
	pseudoRoot := Value{Type: ValPair, PairRight: &Value{Type: ValNull}}
	lastPair := &pseudoRoot
	for {
		if lastPair.Type != ValPair {
			panic("Not ValPair when it has to be")
		}
		value := Value{Type: ValPair}
		if expression.Left != nil {
			left, err:= self.Eval(*expression.Left)
			if err != nil {
				return *pseudoRoot.PairRight, err
			}
			value.PairLeft = &left
		}
		lastPair.PairRight = &value
		if expression.Right == nil {
			return *pseudoRoot.PairRight, nil
		}
		if IsNilExpression(*expression.Right) {
			return *pseudoRoot.PairRight, nil
		}
		if expression.Right.Atom.Type != AtomInvalid {
			right, err:= self.Eval(*expression.Right)
			value.PairRight = &right
			return *pseudoRoot.PairRight, err
		}
		lastPair = &value
		expression = *expression.Right
	}
}

func (self *Interp) Eval(expression Expression) (Value, error) {
	switch expression.Atom.Type {
	case AtomIdentifier:
		if "#f" == expression.Atom.Representation {
			return Value{Type: ValBool, Bool: false}, nil
		}
		if "#t" == expression.Atom.Representation {
			return Value{Type: ValBool, Bool: true}, nil
		}
		value, ok := self.Table[expression.Atom.Representation]
		if ok == false {
			return Value{Type: ValNull}, NewError(
				self.Source.String(),
				expression.Atom.Offset,
				fmt.Sprintf("Unbound variable \"%v\"", expression.Atom.Representation))
		}
		return value, nil
	case AtomNumber:
		number, err := strconv.Atoi(expression.Atom.Representation)
		if err != nil {
			return Value{Type: ValNull}, NewError(
				self.Source.String(),
				expression.Atom.Offset,
				fmt.Sprintf("Can't parse number %v", expression.Atom.Representation))
		}
		return Value{Type: ValNumber, Number: number}, nil
	case AtomString:
		return Value{Type: ValString, StringData: expression.Atom.Representation}, nil
	case AtomInvalid:
		left, err := self.Eval(*expression.Left)
		if err != nil {
			return Value{Type: ValNull}, err
		}
		if left.Type == ValProc {
			right, err := self.EvalRight(*expression.Right)
			if err != nil {
				return Value{Type: ValNull}, err
			}
			return left.Proc(right, *self)
		}
	}
	panic(fmt.Sprintf("Unimplemented atom type evaluation %v", expression.Atom))
}

func TestLex() {
	var lex Lex
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
		if c[0] == '\n' && len(tokens) > 0 {
			fmt.Println(TokensFormatter{lex.Source.String(), tokens})
			tokens = tokens[:0]
		}
	}
}

func TestPars() {
	var parser Pars
	for {
		expression, err := parser.ParseNext(os.Stdin)
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
		fmt.Printf("Expression: %v\n", expression)
	}
}

func TestEval() {
	plusFn := func(arg Value, interp Interp) (Value, error) {
		var acc int
		for {
			if arg.Type != ValPair {
				return Value{Type: ValNull}, NewError(
					interp.Source.String(),
					0, // TODO pass offset
					fmt.Sprintf(
						"Unexpected arg carrier type %v, expected ValPair",
						arg.Type))
			}
			if arg.PairLeft == nil {
				return Value{Type: ValNull}, NewError(
					interp.Source.String(),
					0, // TODO pass offset
					fmt.Sprintf("Unexpected value type ValNull, expected ValPair"))
			}
			left := *arg.PairLeft
			if left.Type != ValNumber {
				return Value{Type: ValNull}, NewError(
					interp.Source.String(),
					0, // TODO pass offset
					fmt.Sprintf(
						"Unexpected value type %v, expected ValNumber",
						arg.Type))
			}
			acc += left.Number
			if arg.PairRight == nil || arg.PairRight.Type == ValNull {
				break
			}
			arg = *arg.PairRight
		}
		return Value{Type: ValNumber, Number: acc}, nil
	}
	var parser Pars
	var interpreter Interp
	interpreter.Source = &parser.Lex.Source
	interpreter.Table = map[string]Value{
		"+": Value{Type: ValProc, Proc: plusFn},
		}
	for {
		expression, err := parser.ParseNext(os.Stdin)
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("Parsing error: %s\n", err.Error())
			continue
		}
		result, err := interpreter.Eval(expression)
		if err != nil {
			fmt.Printf("Eval error: %s\n", err.Error())
		}
		fmt.Printf("Eval result: %v\n", result)
	}
}

func main() {
	TestEval()
}
