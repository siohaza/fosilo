package mapmeta

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type valueKind int

const (
	valueNull valueKind = iota
	valueNumber
	valueString
	valueBool
	valueList
	valueDict
)

type Value struct {
	kind valueKind
	num  float64
	str  string
	bval bool
	list []Value
	dict map[string]Value
}

func numberValue(f float64) Value {
	return Value{kind: valueNumber, num: f}
}

func stringValue(s string) Value {
	return Value{kind: valueString, str: s}
}

func boolValue(b bool) Value {
	return Value{kind: valueBool, bval: b}
}

func listValue(values []Value) Value {
	return Value{kind: valueList, list: values}
}

func dictValue(values map[string]Value) Value {
	return Value{kind: valueDict, dict: values}
}

func nullValue() Value {
	return Value{kind: valueNull}
}

func (v Value) asNumber() (float64, error) {
	switch v.kind {
	case valueNumber:
		return v.num, nil
	case valueBool:
		if v.bval {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("value is not numeric")
	}
}

func (v Value) asString() (string, error) {
	if v.kind == valueString {
		return v.str, nil
	}
	return "", fmt.Errorf("value is not a string")
}

func (v Value) asBool() (bool, error) {
	if v.kind == valueBool {
		return v.bval, nil
	}
	return false, fmt.Errorf("value is not a bool")
}

func (v Value) isNumber() bool {
	return v.kind == valueNumber
}

func (v Value) isInt() bool {
	if v.kind != valueNumber {
		return false
	}
	_, frac := math.Modf(v.num)
	return math.Abs(frac) < 1e-9
}

func (v Value) index(idx Value) (Value, error) {
	i, err := idx.asNumber()
	if err != nil {
		return Value{}, err
	}
	index := int(i)

	switch v.kind {
	case valueList:
		if index < 0 || index >= len(v.list) {
			return Value{}, fmt.Errorf("index out of range")
		}
		return v.list[index], nil
	default:
		return Value{}, fmt.Errorf("value is not indexable")
	}
}

func (v Value) toInterface() any {
	switch v.kind {
	case valueNull:
		return nil
	case valueNumber:
		if v.isInt() {
			return int64(v.num)
		}
		return v.num
	case valueString:
		return v.str
	case valueBool:
		return v.bval
	case valueList:
		return convertList(v.list)
	case valueDict:
		result := make(map[string]any, len(v.dict))
		for k, val := range v.dict {
			if converted := val.toInterface(); converted != nil {
				result[k] = converted
			}
		}
		return result
	default:
		return nil
	}
}

func convertList(list []Value) any {
	if len(list) == 0 {
		return []any{}
	}

	allInts := true
	allFloats := true
	allStrings := true

	for _, val := range list {
		if !val.isNumber() {
			allInts = false
			allFloats = false
		} else if !val.isInt() {
			allInts = false
		}

		if val.kind != valueString {
			allStrings = false
		}
	}

	if allInts {
		out := make([]int64, len(list))
		for i, val := range list {
			out[i] = int64(val.num)
		}
		return out
	}

	if allFloats {
		out := make([]float64, len(list))
		for i, val := range list {
			out[i] = val.num
		}
		return out
	}

	if allStrings {
		out := make([]string, len(list))
		for i, val := range list {
			out[i] = val.str
		}
		return out
	}

	out := make([]any, len(list))
	for i, val := range list {
		out[i] = val.toInterface()
	}
	return out
}

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenIdentifier
	tokenNumber
	tokenString
	tokenSymbol
)

type token struct {
	typ tokenType
	lit string
}

type lexer struct {
	input string
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: input}
}

func (l *lexer) nextToken() token {
	l.skipWhitespace()
	if l.pos >= len(l.input) {
		return token{typ: tokenEOF}
	}

	ch := l.input[l.pos]

	if isIdentifierStart(ch) {
		start := l.pos
		l.pos++
		for l.pos < len(l.input) && isIdentifierPart(l.input[l.pos]) {
			l.pos++
		}
		return token{typ: tokenIdentifier, lit: l.input[start:l.pos]}
	}

	if ch == '.' || unicode.IsDigit(rune(ch)) {
		start := l.pos
		l.pos++
		for l.pos < len(l.input) && (unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '.') {
			l.pos++
		}
		return token{typ: tokenNumber, lit: l.input[start:l.pos]}
	}

	if ch == '\'' || ch == '"' {
		return l.scanString(ch)
	}

	if ch == '/' && l.peekNext('/') {
		l.pos += 2
		return token{typ: tokenSymbol, lit: "//"}
	}

	l.pos++
	return token{typ: tokenSymbol, lit: string(ch)}
}

func (l *lexer) scanString(delim byte) token {
	l.pos++ // consume opening quote
	start := l.pos
	var builder strings.Builder
	escaped := false

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if escaped {
			switch ch {
			case 'n':
				builder.WriteByte('\n')
			case 't':
				builder.WriteByte('\t')
			case '\\':
				builder.WriteByte('\\')
			case '\'':
				builder.WriteByte('\'')
			case '"':
				builder.WriteByte('"')
			default:
				builder.WriteByte(ch)
			}
			escaped = false
		} else {
			if ch == '\\' {
				escaped = true
				l.pos++
				continue
			}
			if ch == delim {
				token := token{typ: tokenString, lit: builder.String()}
				l.pos++
				return token
			}
			builder.WriteByte(ch)
		}
		l.pos++
	}

	return token{typ: tokenString, lit: l.input[start:]}
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		switch l.input[l.pos] {
		case ' ', '\t', '\n', '\r':
			l.pos++
		default:
			return
		}
	}
}

func (l *lexer) peekNext(ch byte) bool {
	return l.pos+1 < len(l.input) && l.input[l.pos+1] == ch
}

type parser struct {
	lex    *lexer
	cur    token
	peeked bool
}

func newParser(input string) *parser {
	return &parser{lex: newLexer(input)}
}

func (p *parser) next() token {
	if p.peeked {
		p.peeked = false
		return p.cur
	}
	p.cur = p.lex.nextToken()
	return p.cur
}

func (p *parser) peek() token {
	if !p.peeked {
		p.cur = p.lex.nextToken()
		p.peeked = true
	}
	return p.cur
}

func (p *parser) consume(expected string) bool {
	tok := p.peek()
	if tok.typ == tokenSymbol && tok.lit == expected {
		p.next()
		return true
	}
	return false
}

func (p *parser) expect(expected string) error {
	if !p.consume(expected) {
		return fmt.Errorf("expected %s", expected)
	}
	return nil
}

func evaluateExpression(expr string, env map[string]Value) (Value, error) {
	parser := newParser(expr)
	val, err := parser.parseExpression(env)
	if err != nil {
		return Value{}, err
	}
	return val, nil
}

func (p *parser) parseExpression(env map[string]Value) (Value, error) {
	return p.parseAdditive(env)
}

func (p *parser) parseAdditive(env map[string]Value) (Value, error) {
	left, err := p.parseMultiplicative(env)
	if err != nil {
		return Value{}, err
	}

	for {
		tok := p.peek()
		if tok.typ != tokenSymbol || (tok.lit != "+" && tok.lit != "-") {
			break
		}
		p.next()
		right, err := p.parseMultiplicative(env)
		if err != nil {
			return Value{}, err
		}
		left, err = applyBinary(tok.lit, left, right)
		if err != nil {
			return Value{}, err
		}
	}

	return left, nil
}

func (p *parser) parseMultiplicative(env map[string]Value) (Value, error) {
	left, err := p.parseUnary(env)
	if err != nil {
		return Value{}, err
	}

	for {
		tok := p.peek()
		if tok.typ != tokenSymbol || (tok.lit != "*" && tok.lit != "/" && tok.lit != "//") {
			break
		}
		p.next()
		right, err := p.parseUnary(env)
		if err != nil {
			return Value{}, err
		}
		left, err = applyBinary(tok.lit, left, right)
		if err != nil {
			return Value{}, err
		}
	}

	return left, nil
}

func (p *parser) parseUnary(env map[string]Value) (Value, error) {
	tok := p.peek()
	if tok.typ == tokenSymbol && (tok.lit == "+" || tok.lit == "-") {
		p.next()
		val, err := p.parseUnary(env)
		if err != nil {
			return Value{}, err
		}
		if tok.lit == "-" {
			num, err := val.asNumber()
			if err != nil {
				return Value{}, err
			}
			return numberValue(-num), nil
		}
		return val, nil
	}
	return p.parsePrimary(env)
}

func (p *parser) parsePrimary(env map[string]Value) (Value, error) {
	tok := p.next()
	switch tok.typ {
	case tokenNumber:
		f, err := strconv.ParseFloat(tok.lit, 64)
		if err != nil {
			return Value{}, err
		}
		val := numberValue(f)
		return p.parseIndexing(val, env)
	case tokenString:
		val := stringValue(tok.lit)
		return p.parseIndexing(val, env)
	case tokenIdentifier:
		switch tok.lit {
		case "True":
			val := boolValue(true)
			return p.parseIndexing(val, env)
		case "False":
			val := boolValue(false)
			return p.parseIndexing(val, env)
		case "None":
			val := nullValue()
			return p.parseIndexing(val, env)
		}
		if val, ok := env[tok.lit]; ok {
			return p.parseIndexing(val, env)
		}
		return Value{}, fmt.Errorf("unknown identifier %s", tok.lit)
	case tokenSymbol:
		switch tok.lit {
		case "(":
			return p.parseTupleOrGroup(env)
		case "[":
			return p.parseList(env)
		case "{":
			return p.parseDict(env)
		default:
			return Value{}, fmt.Errorf("unexpected symbol %s", tok.lit)
		}
	default:
		return Value{}, fmt.Errorf("unexpected token %v", tok)
	}
}

func (p *parser) parseTupleOrGroup(env map[string]Value) (Value, error) {
	if p.consume(")") {
		return listValue(nil), nil
	}

	first, err := p.parseExpression(env)
	if err != nil {
		return Value{}, err
	}

	if !p.consume(",") {
		if err := p.expect(")"); err != nil {
			return Value{}, err
		}
		return first, nil
	}

	values := []Value{first}
	for {
		if p.consume(")") {
			break
		}
		val, err := p.parseExpression(env)
		if err != nil {
			return Value{}, err
		}
		values = append(values, val)
		if p.consume(")") {
			break
		}
		if !p.consume(",") {
			if err := p.expect(")"); err != nil {
				return Value{}, err
			}
			break
		}
	}
	return listValue(values), nil
}

func (p *parser) parseList(env map[string]Value) (Value, error) {
	values := []Value{}
	if p.consume("]") {
		return listValue(values), nil
	}

	for {
		if tok := p.peek(); tok.typ == tokenSymbol && tok.lit == "]" {
			p.next()
			break
		}
		val, err := p.parseExpression(env)
		if err != nil {
			return Value{}, err
		}
		values = append(values, val)

		if p.consume("]") {
			break
		}
		if !p.consume(",") {
			if err := p.expect("]"); err != nil {
				return Value{}, err
			}
			break
		}
	}
	return listValue(values), nil
}

func (p *parser) parseDict(env map[string]Value) (Value, error) {
	values := make(map[string]Value)
	if p.consume("}") {
		return dictValue(values), nil
	}

	for {
		if tok := p.peek(); tok.typ == tokenSymbol && tok.lit == "}" {
			p.next()
			break
		}
		keyTok := p.next()
		key := ""
		switch keyTok.typ {
		case tokenIdentifier:
			key = keyTok.lit
		case tokenString:
			key = keyTok.lit
		default:
			return Value{}, fmt.Errorf("invalid dict key")
		}

		if err := p.expect(":"); err != nil {
			return Value{}, err
		}
		val, err := p.parseExpression(env)
		if err != nil {
			return Value{}, err
		}
		values[key] = val

		if p.consume("}") {
			break
		}
		if !p.consume(",") {
			if err := p.expect("}"); err != nil {
				return Value{}, err
			}
			break
		}
	}
	return dictValue(values), nil
}

func (p *parser) parseIndexing(base Value, env map[string]Value) (Value, error) {
	for p.consume("[") {
		idx, err := p.parseExpression(env)
		if err != nil {
			return Value{}, err
		}
		if err := p.expect("]"); err != nil {
			return Value{}, err
		}
		base, err = base.index(idx)
		if err != nil {
			return Value{}, err
		}
	}
	return base, nil
}

func applyBinary(op string, left, right Value) (Value, error) {
	l, err := left.asNumber()
	if err != nil {
		return Value{}, err
	}
	r, err := right.asNumber()
	if err != nil {
		return Value{}, err
	}

	switch op {
	case "+":
		return numberValue(l + r), nil
	case "-":
		return numberValue(l - r), nil
	case "*":
		return numberValue(l * r), nil
	case "/":
		return numberValue(l / r), nil
	case "//":
		return numberValue(math.Floor(l / r)), nil
	default:
		return Value{}, fmt.Errorf("unsupported operator %s", op)
	}
}
