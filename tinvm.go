package tinvm

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type TinVM struct {
	source      string
	pc          int
	variables   map[string]interface{}
	customFuncs map[string]func(*TinVM, []interface{}) error
	returnFlag  bool
}

func New() *TinVM {
	vm := &TinVM{
		pc:          0,
		variables:   make(map[string]interface{}),
		customFuncs: make(map[string]func(*TinVM, []interface{}) error),
		returnFlag:  false,
	}

	vm.AddFunction("print", customFunc_Print)
	vm.AddFunction("println", customFunc_Println)
	vm.AddFunction("wait", customFunction_Wait)

	return vm
}

func (vm *TinVM) Run(source string, filename string) {
	// Preprocess the source code to handle imports
	preprocessedSource, err := vm.preprocess(string(source)+"\000", filename)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Add initial filename directive
	vm.source = fmt.Sprintf("#%s:%d\n%s", filename, 1, preprocessedSource)

	active := true

	for vm.nextChar() != '\000' {
		if vm.source[vm.pc] == '#' {
			vm.pc++
			for vm.source[vm.pc] != '\n' && vm.source[vm.pc] != '\000' {
				vm.pc++
			}
			vm.pc++ // Move past the newline
		} else {
			vm.block(active)
		}
	}
}

func (vm *TinVM) AddFunction(name string, fn func(*TinVM, []interface{}) error) {
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
		b = e.value.(float64) != 0 // Converting integer to boolean
		vm.nextChar()
		if vm.takeString("==") {
			b = ((int)(e.value.(float64)) == (int)(vm.mathExpression(active)))
		} else if vm.takeString("!=") {
			b = ((int)(e.value.(float64)) != (int)(vm.mathExpression(active)))
		} else if vm.takeString("<=") {
			b = ((int)(e.value.(float64)) <= (int)(vm.mathExpression(active)))
		} else if vm.takeString("<") {
			b = ((int)(e.value.(float64)) < (int)(vm.mathExpression(active)))
		} else if vm.takeString(">=") {
			b = ((int)(e.value.(float64)) >= (int)(vm.mathExpression(active)))
		} else if vm.takeString(">") {
			b = ((int)(e.value.(float64)) > (int)(vm.mathExpression(active)))
		}
	case 'f':
		b = e.value.(float64) != 0.0
		vm.nextChar()
		if vm.takeString("==") {
			b = (e.value.(float64) == vm.mathExpression(active))
		} else if vm.takeString("!=") {
			b = (e.value.(float64) != vm.mathExpression(active))
		} else if vm.takeString("<=") {
			b = (e.value.(float64) <= vm.mathExpression(active))
		} else if vm.takeString("<") {
			b = (e.value.(float64) < vm.mathExpression(active))
		} else if vm.takeString(">=") {
			b = (e.value.(float64) >= vm.mathExpression(active))
		} else if vm.takeString(">") {
			b = (e.value.(float64) > vm.mathExpression(active))
		}
	case 's':
		b = e.value.(string) != ""
		vm.nextChar()
		if vm.takeString("==") {
			b = (e.value.(string) == vm.stringExpression())
		} else if vm.takeString("!=") {
			b = (e.value.(string) != vm.stringExpression())
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

func (vm *TinVM) mathFactor(active bool) float64 {
	m := 0.0
	if vm.takeNext('(') {
		m = vm.mathExpression(active)
		if !vm.takeNext(')') {
			vm.error("missing ')'")
		}
	} else if isDigit(vm.nextChar()) {
		numStr := ""
		for isDigit(vm.look()) || vm.look() == '.' {
			numStr += string(vm.take())
		}
		if active {
			var err error
			m, err = strconv.ParseFloat(numStr, 64)
			if err != nil {
				vm.error("invalid number format")
			}
		}
	} else {
		ident := vm.takeNextAlnum()
		if val, ok := vm.variables[ident]; ok {
			switch value := val.(type) {
			case int:
				m = float64(value)
			case float64:
				m = value
			default:
				vm.error("unknown variable")
			}
		} else {
			vm.error("unknown variable")
		}
	}
	return m
}

func (vm *TinVM) mathTerm(active bool) float64 {
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

func (vm *TinVM) mathExpression(active bool) float64 {
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

func (vm *TinVM) parseString() string {
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
	} else {
		ident := vm.takeNextAlnum()
		if ident != "" {
			if val, ok := vm.variables[ident]; ok {
				switch value := val.(type) {
				case string:
					s = value
				case int:
					s = strconv.Itoa(value)
				case float64:
					s = strconv.FormatFloat(value, 'f', -1, 64)
				default:
					vm.error("unknown variable type")
				}
			} else {
				vm.error("unknown variable")
			}
		} else {
			// Handle direct numeric values
			if isDigit(vm.nextChar()) || vm.nextChar() == '-' {
				numStr := ""
				for isDigit(vm.look()) || vm.look() == '.' || vm.look() == '-' {
					numStr += string(vm.take())
				}
				if val, err := strconv.ParseFloat(numStr, 64); err == nil {
					s = strconv.FormatFloat(val, 'f', -1, 64)
				} else {
					vm.error("invalid number format")
				}
			} else {
				vm.error("expected string or number")
			}
		}
	}
	return s
}

func (vm *TinVM) stringExpression() string {
	s := vm.parseString()
	for vm.takeNext('+') {
		s += vm.parseString() // String concatenation
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
		return Expression{'s', vm.stringExpression()}
	}

	val, ok := vm.variables[ident]
	if ok && val != nil && fmt.Sprintf("%T", val) == "string" {
		return Expression{'s', vm.stringExpression()}
	}

	// Check for numeric expression
	expr := vm.mathExpression(active)

	// does it have decimal places?
	if hasDecimalPlaces(expr) {
		return Expression{'f', expr}
	} else {
		return Expression{'i', expr}
	}

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

// Preprocess the script to handle imports and preserve import directives with line numbers
func (vm *TinVM) preprocess(source string, filename string) (string, error) {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(source))
	lineNumber := 1

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

			// Append the import directive and the content of the imported file to the result
			result.WriteString(fmt.Sprintf("#%s:%d\n", importPath, 1))
			importedContent, err := vm.preprocess(string(content), importPath)
			if err != nil {
				return "", err
			}
			result.WriteString(importedContent)
			result.WriteString(fmt.Sprintf("#%s:%d\n", filename, lineNumber+1))
		} else {
			result.WriteString(lineRaw + "\n")
		}
		lineNumber++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading source: %v", err)
	}

	return result.String(), nil
}

func (vm *TinVM) error(text string) {
	vm.errorWithPosition(text, vm.pc)
}

func (vm *TinVM) errorWithPosition(text string, pc int) {
	// Find the current file and line number
	currentFile := "unknown"
	currentLine := 1

	for i := pc; i >= 0; i-- {
		if vm.source[i] == '#' {
			startDirective := i + 1
			endDirective := strings.Index(vm.source[startDirective:], "\n")
			if endDirective == -1 {
				endDirective = len(vm.source)
			} else {
				endDirective += startDirective
			}
			directive := strings.TrimSpace(vm.source[startDirective:endDirective])
			parts := strings.Split(directive, ":")
			if len(parts) == 2 {
				currentFile = parts[0]
				lineNumber, _ := strconv.Atoi(parts[1])
				linesBeforeError := strings.Count(vm.source[i:pc], "\n") - 1
				currentLine = lineNumber + linesBeforeError
				break
			}
		}
	}

	start := strings.LastIndex(vm.source[:pc], "\n") + 1
	end := strings.Index(vm.source[pc:], "\n")
	if end == -1 {
		end = len(vm.source)
	} else {
		end += pc
	}

	fmt.Printf("\nERROR %s in '%s' on line %d: '%s_%s'\n", text, currentFile, currentLine, vm.source[start:pc], vm.source[pc:end])
	os.Exit(1)
}

func (vm *TinVM) handleCustomFunction(ident string, active bool) bool {
	if fn, ok := vm.customFuncs[ident]; ok {
		// Collect arguments for the custom function without parentheses
		args, startPc := vm.collectArgs(active)
		if active {
			err := fn(vm, args)
			if err != nil {
				vm.errorWithPosition(fmt.Sprintf("error in function '%s': %v", ident, err), startPc)
			}
		}
		return true
	}
	return false
}

func (vm *TinVM) collectArgs(active bool) ([]interface{}, int) {
	var args []interface{}
	startPc := vm.pc

	for vm.nextChar() != '\n' && vm.nextChar() != '\000' {
		e := vm.expression(active)
		if active {
			switch e.typ {
			case 's':
				if strVal, ok := e.value.(string); ok {
					args = append(args, strVal)
				} else {
					panic("Type assertion to string failed in collectArgs")
				}
			case 'i':
				// Internally, all numbers are treated as float64
				if floatVal, ok := e.value.(float64); ok {
					args = append(args, int(floatVal))
				} else {
					panic("Type assertion to int failed in collectArgs for int case")
				}
			case 'f':
				if floatVal, ok := e.value.(float64); ok {
					args = append(args, floatVal)
				} else {
					panic("Type assertion to float64 failed in collectArgs for float case")
				}
			default:
				panic("Unknown type in collectArgs: this should never happen!")
			}
		}
		if vm.nextChar() == ',' {
			vm.take()
		} else {
			break
		}
	}
	return args, startPc
}

func hasDecimalPlaces(f float64) bool {
	return f != math.Floor(f)
}
