package tinvm

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type TinVM struct {
	source      string
	pc          int
	variables   map[string]interface{}
	customFuncs map[string]func([]interface{}) error
	returnFlag  bool
}

func New() *TinVM {
	vm := &TinVM{
		pc:          0,
		variables:   make(map[string]interface{}),
		customFuncs: make(map[string]func([]interface{}) error),
		returnFlag:  false,
	}

	vm.AddFunction("print", customPrintFunction)

	return vm
}

func (vm *TinVM) Run(source string) {
	preprocessedSource, err := vm.preprocess(source + "\000")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}

	vm.source = preprocessedSource

	active := true
	for vm.nextChar() != '\000' {
		vm.block(active)
	}
}

func (vm *TinVM) AddFunction(name string, fn func([]interface{}) error) {
	vm.customFuncs[name] = fn
}

func (vm *TinVM) AddVariable(name string, value interface{}) {
	vm.variables[name] = value
}

func (vm *TinVM) look() byte {
	if vm.source[vm.pc] == ';' {
		for vm.source[vm.pc] != '\n' && vm.source[vm.pc] != '\000' {
			vm.pc++
		}
	}
	return vm.source[vm.pc]
}

func (vm *TinVM) take() byte {
	c := vm.look()
	vm.pc++
	return c
}

func (vm *TinVM) takeString(word string) bool {
	originalPc := vm.pc
	for _, c := range word {
		if vm.take() != byte(c) {
			vm.pc = originalPc
			return false
		}
	}
	return true
}

func (vm *TinVM) nextChar() byte {
	for vm.look() == ' ' || vm.look() == '\t' || vm.look() == '\n' || vm.look() == '\r' {
		vm.take()
	}
	return vm.look()
}

func (vm *TinVM) takeNext(char byte) bool {
	if vm.nextChar() == char {
		vm.take()
		return true
	}
	return false
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func isAlpha(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

func isAlnum(c byte) bool {
	return isDigit(c) || isAlpha(c)
}

func isAddOp(c byte) bool {
	return c == '+' || c == '-'
}

func isMulOp(c byte) bool {
	return c == '*' || c == '/'
}

func (vm *TinVM) takeNextAlnum() string {
	alnum := ""
	if isAlpha(vm.nextChar()) {
		for isAlnum(vm.look()) {
			alnum += string(vm.take())
		}
	}
	return alnum
}

func (vm *TinVM) booleanFactor(active bool) bool {
	inv := vm.takeNext('!')
	e := vm.expression(active)
	var b bool

	switch e.typ {
	case 'i':
		b = e.value.(int) != 0 // Converting integer to boolean
		vm.nextChar()
		if vm.takeString("==") {
			b = (e.value.(int) == vm.mathExpression(active))
		} else if vm.takeString("!=") {
			b = (e.value.(int) != vm.mathExpression(active))
		} else if vm.takeString("<=") {
			b = (e.value.(int) <= vm.mathExpression(active))
		} else if vm.takeString("<") {
			b = (e.value.(int) < vm.mathExpression(active))
		} else if vm.takeString(">=") {
			b = (e.value.(int) >= vm.mathExpression(active))
		} else if vm.takeString(">") {
			b = (e.value.(int) > vm.mathExpression(active))
		}
	case 's':
		b = e.value.(string) != ""
		vm.nextChar()
		if vm.takeString("==") {
			b = (e.value.(string) == vm.stringExpression(active))
		} else if vm.takeString("!=") {
			b = (e.value.(string) != vm.stringExpression(active))
		}
	}

	return active && (b != inv) // Always returns False if inactive
}

func (vm *TinVM) booleanTerm(active bool) bool {
	b := vm.booleanFactor(active)
	for vm.takeString("and") {
		nextFactor := vm.booleanFactor(active)
		b = b && nextFactor
	}
	return b
}

func (vm *TinVM) booleanExpression(active bool) bool {
	b := vm.booleanTerm(active)
	for vm.takeString("or") {
		nextTerm := vm.booleanTerm(active)
		b = b || nextTerm
	}
	return b
}

func (vm *TinVM) mathFactor(active bool) int {
	m := 0
	if vm.takeNext('(') {
		m = vm.mathExpression(active)
		if !vm.takeNext(')') {
			vm.error("missing ')'")
		}
	} else if isDigit(vm.nextChar()) {
		for isDigit(vm.look()) {
			m = 10*m + int(vm.take()-'0')
		}
	} else if vm.takeString("val(") {
		s := vm.string(active)
		if active {
			if v, err := strconv.Atoi(s); err == nil {
				m = v
			}
		}
		if !vm.takeNext(')') {
			vm.error("missing ')'")
		}
	} else {
		ident := vm.takeNextAlnum()
		if val, ok := vm.variables[ident]; ok {
			if value, valid := val.(int); valid {
				m = value
			} else {
				vm.error("unknown variable")
			}
		} else {
			vm.error("unknown variable")
		}
	}
	return m
}

func (vm *TinVM) mathTerm(active bool) int {
	m := vm.mathFactor(active)
	for isMulOp(vm.nextChar()) {
		c := vm.take()
		m2 := vm.mathFactor(active)
		if c == '*' {
			m *= m2 // Multiplication
		} else {
			m /= m2 // Division
		}
	}
	return m
}

func (vm *TinVM) mathExpression(active bool) int {
	c := vm.nextChar() // Check for an optional leading sign
	if isAddOp(c) {
		c = vm.take()
	}
	m := vm.mathTerm(active)
	if c == '-' {
		m = -m
	}
	for isAddOp(vm.nextChar()) {
		c = vm.take()
		m2 := vm.mathTerm(active)
		if c == '+' {
			m += m2 // Addition
		} else {
			m -= m2 // Subtraction
		}
	}
	return m
}

func (vm *TinVM) string(active bool) string {
	s := ""
	if vm.takeNext('"') { // Literal string
		for !vm.takeString("\"") {
			if vm.look() == '\000' {
				vm.error("unexpected EOF")
			}
			if vm.takeString("\\n") {
				s += "\n"
			} else {
				s += string(vm.take())
			}
		}
	} else if vm.takeString("str(") { // str(...)
		s = strconv.Itoa(vm.mathExpression(active))
		if !vm.takeNext(')') {
			vm.error("missing ')'")
		}
	} else if vm.takeString("input()") {
		if active {
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			s = strings.TrimSpace(input)
		}
	} else {
		ident := vm.takeNextAlnum()
		if val, ok := vm.variables[ident]; ok {
			if value, valid := val.(string); valid {
				s = value
			} else {
				vm.error("not a string")
			}
		} else {
			vm.error("not a string")
		}
	}
	return s
}

func (vm *TinVM) stringExpression(active bool) string {
	s := vm.string(active)
	for vm.takeNext('+') {
		s += vm.string(active) // String concatenation
	}
	return s
}

type Expression struct {
	typ   byte
	value interface{}
}

func (vm *TinVM) expression(active bool) Expression {
	originalPc := vm.pc
	ident := vm.takeNextAlnum()
	vm.pc = originalPc // Scan for identifier or "str"

	nextChar := vm.nextChar()
	if nextChar == '"' || ident == "str" || ident == "input" {
		return Expression{'s', vm.stringExpression(active)}
	}

	val, ok := vm.variables[ident]
	if ok && val != nil && fmt.Sprintf("%T", val) == "string" {
		return Expression{'s', vm.stringExpression(active)}
	}

	return Expression{'i', vm.mathExpression(active)}
}

func (vm *TinVM) doWhile(active bool) {
	localActive := active
	pcWhile := vm.pc // Save PC of the while statement
	for vm.booleanExpression(localActive) {
		vm.block(localActive)
		if vm.returnFlag {
			return
		}
		vm.pc = pcWhile
	}
	vm.block(false) // Scan over inactive block and leave while
}

func (vm *TinVM) doIfElse(active bool) {
	b := vm.booleanExpression(active)
	if active && b {
		vm.block(active) // Process if block
	} else {
		vm.block(false)
	}
	vm.nextChar()
	if vm.takeString("else") { // Process else block
		if active && !b {
			vm.block(active)
		} else {
			vm.block(false)
		}
	}
}

func (vm *TinVM) doCall(active bool) {
	ident := vm.takeNextAlnum()
	if val, ok := vm.variables[ident]; ok && fmt.Sprintf("%T", val) == "func(bool)" {
		vm.variables[ident].(func(bool))(active)
	} else {
		vm.error("unknown subroutine")
	}
}

func (vm *TinVM) doDefDecl() {
	ident := vm.takeNextAlnum()
	if ident == "" {
		vm.error("missing subroutine identifier")
	}
	pc := vm.pc
	vm.variables[ident] = func(active bool) {
		retPc := vm.pc
		vm.pc = pc
		vm.block(active)
		vm.pc = retPc
	}
	vm.block(false)
}

func (vm *TinVM) doAssign(active bool) {
	ident := vm.takeNextAlnum()
	if !vm.takeNext('=') || ident == "" {
		vm.error("unknown statement")
	}
	e := vm.expression(active)
	if active || vm.variables[ident] == nil {
		vm.variables[ident] = e.value // Initialize variable even if block is inactive
	}
}

func (vm *TinVM) doBreak(active bool) {
	if active {
		active = false // Switch off execution within enclosing loop
	}
}

func (vm *TinVM) doReturn(active bool) {
	if active {
		vm.returnFlag = true
	}
}

func (vm *TinVM) statement(active bool) {
	ident := vm.takeNextAlnum()
	if ident == "" {
		vm.error("unknown statement")
	}

	// Check if the statement is a custom function
	if vm.handleCustomFunction(ident, active) {
		return
	}

	// Reset pc to allow normal statement parsing if not a custom function
	vm.pc -= len(ident)
	switch {
	case vm.takeString("if"):
		vm.doIfElse(active)
	case vm.takeString("while"):
		vm.doWhile(active)
	case vm.takeString("break"):
		vm.doBreak(active)
	case vm.takeString("call"):
		vm.doCall(active)
	case vm.takeString("def"):
		vm.doDefDecl()
	case vm.takeString("return"):
		vm.doReturn(active)
	default:
		vm.doAssign(active)
	}
}

func (vm *TinVM) block(active bool) {
	if vm.takeNext('{') {
		for !vm.takeNext('}') {
			if vm.returnFlag {
				return
			}
			vm.statement(active)
		}
	} else {
		vm.statement(active)
	}
}

// Preprocess the script to handle imports
func (vm *TinVM) preprocess(source string) (string, error) {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(source))

	for scanner.Scan() {
		lineRaw := scanner.Text()
		line := strings.TrimSpace(lineRaw)
		if strings.HasPrefix(line, "#import ") {
			// Extract the file path from the import statement
			importPath := strings.TrimSpace(line[len("#import "):])
			importPath = strings.Trim(importPath, "\"") + ".tin"

			// Read the content of the file
			content, err := os.ReadFile(importPath)
			if err != nil {
				return "", fmt.Errorf("error reading file %s: %v", importPath, err)
			}

			// Append the content of the imported file to the result
			result.WriteString(string(content) + "\n")
		} else {
			result.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading source: %v", err)
	}

	return result.String(), nil
}

func (vm *TinVM) error(text string) {
	s := strings.LastIndex(vm.source[:vm.pc], "\n") + 1
	e := strings.Index(vm.source[vm.pc:], "\n")
	if e == -1 {
		e = len(vm.source)
	} else {
		e += vm.pc
	}
	fmt.Printf("\nERROR %s in line %d: '%s_%s'\n", text, strings.Count(vm.source[:vm.pc], "\n")+1, vm.source[s:vm.pc], vm.source[vm.pc:e])
	os.Exit(1)
}

func (vm *TinVM) handleCustomFunction(ident string, active bool) bool {
	if fn, ok := vm.customFuncs[ident]; ok {
		// Collect arguments for the custom function without parentheses
		args := vm.collectArgs(active)
		if active {
			err := fn(args)
			if err != nil {
				vm.error(fmt.Sprintf("error in function '%s': %v", ident, err))
			}
		}
		return true
	}
	return false
}

func (vm *TinVM) collectArgs(active bool) []interface{} {
	var args []interface{}
	for vm.nextChar() != '\n' && vm.nextChar() != '\000' {
		e := vm.expression(active)
		if active {
			args = append(args, e.value)
		}
		if vm.nextChar() == ',' {
			vm.take()
		} else {
			break
		}
	}
	return args
}
