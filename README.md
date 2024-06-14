
# TinVM: A Tiny Scripting Language

TinVM is a tiny interpreter for the custom scripting language "tin".

Tin is basically one step above the BASIC language.

Tin is written in pure Go ~600 loc.

Tin is:
- tiny
- easy
- embeddable
- extendable via custom functions and variables

## Using the Language

### Writing Scripts

Tin scripts have the `.tin` extension. Here is a sample script (`test.tin`):

```tin
; the following line would import the file "bye.tin"
; an import is just a copy what is in the file and replace
; the import statement with the content of the file
; so no namespacing or double import check
; in tin it is ideomatic to have a main.tin that import everything once
#import bye
def demo {
    a = "math expressions "
    b = (10 + 1*2 + (4 + 5)*2 )*3 - 111
    print "Test " + a + b, "\n"
    if 0 or 5 > -2 or 0 print "True\n" else print "False\n"
    x=0 
    while x<20 {
        if !x<10 break
        print "." x=x+1
    }
    print "\n"

    if 10 == 20 {
        print "10 is 20\n"
    } else if 10 != 20 {
        print "10 is not 20\n"
    } else {
        print "i am confused\n"
    }
}

def testReturn {
    print "Before return \n"
    return
    print "after return \n"
}



print "Hello, World!\n"     ; program entry point

pi = 3.14
print pi , "\n"

pi2 = pi * pi
print pi2, "\n"

call demo
call testReturn
call printBye
```

### Running Scripts

To run a script, use the following command:

```
./tin.exe path/to/your/script.tin
```

## Embedding TinVM in Your Project

To embed TinVM in your own Go project, follow these steps:

1. Add `tinvm` to your project:
    ```
    go get github.com/RednibCoding/tinvm
    ```

2. Import TinVM in your Go code:
    ```go
    import "github.com/RednibCoding/tinvm"
    ```

### Example Usage

Here is an example of how to use TinVM in your Go project:

```go
package main

import (
	"fmt"
	"os"
	"github.com/RednibCoding/tinvm"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("USAGE: tin <sourcefile>")
		os.Exit(1)
	}
	source, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Printf("ERROR: Can't find source file '%s'.\n", args[1])
		os.Exit(1)
	}

	vm := tinvm.New()
	vm.Run(string(source), args[1])
}
```

## Defining Custom Functions and Variables

### Custom Functions

Custom functions can be defined and added to the VM using the `AddFunction` method. Custom functions must have the following signature:

```go
func(vm *TinVM, args []interface{}) error
```

Example:

```go
func customPrintFunction(vm *tinvm.TinVM, args []interface{}) error {
    for _, arg := range args {
        switch v := arg.(type) {
        case string:
            fmt.Print(v)
        case int:
            fmt.Print(v)
        case float64:
            fmt.Print(v)
        default:
            return fmt.Errorf("unsupported argument type")
        }
    }
    fmt.Println()
    return nil
}

vm.AddFunction("print", customPrintFunction)

```
>**Note:** It is important to always check the number of arguments and their types, as you don't know what errors users might make in their scripts:
```go

func customFunction_Move(vm *tinvm.TinVM, args []interface{}) error {
    // Check the number of arguments (assuming 2 are expected here)
	if len(args) != 2 {
		return fmt.Errorf("move requires exactly 2 arguments")
	}

	// Using type assertions to check if x and y are of type int
	x, ok1 := args[0].(int)
	y, ok2 := args[1].(int)

	if !ok1 || !ok2 {
		return fmt.Errorf("both arguments must be of type int")
	}

	fmt.Printf("x: %d, y: %d\n", x, y)

	Mover.Move(x, y)
	return nil
}
```

### Custom Variables

Custom variables can be defined and added to the VM using the `AddVariable` method. Variables can be of type `string`, `int` or `float64`.

Example:

```go
vm.AddVariable("greeting", "Hello from VM!")
```

## Using the `#import` Directive

The `#import` directive allows you to include the contents of another file in your script. This is useful for modularizing your code. The import statement is simply replaced by the content of the imported file. There is no namespacing or double import check.

Example:

If you have a file `bye.tin`:
```tin
print "Goodbye, World!\n"
```

And your main script `test.tin`:
```tin
#import bye
print "Hello, World!\n"
```

When `test.tin` is run, it will behave as if its content is:
```tin
print "Goodbye, World!\n"
print "Hello, World!\n"
```

## Error Handling

If a custom function returns an error, the interpreter will report it with the line and column number where the error occurred.

Example:

```go
func customPrintFunction(args []interface{}) error {
    for _, arg := range args {
        if arg == "panic" {
            return fmt.Errorf("intentional panic triggered")
        }
        fmt.Print(arg)
    }
    return nil
}
```

If the `print` function encounters the string `"panic"`, it will return an error, and the interpreter will handle and report it.

```
print "panic"
```
output:
```
ERROR error in custom function 'print': intentional panic triggered in line 1: '_print "panic"'
```

# Tin Language Specification

The Tin language is a simple, dynamic scripting language with a single global scope. The following chapter describes the syntax and features of the Tin language, including how to define and call subroutines, use variables, control flow with `if` and `while` statements, data types and falsy values.

## Defining and Calling Subroutines

### Defining Subroutines

Subroutines in Tin are defined using the `def` keyword followed by the subroutine name and a block of code enclosed in curly braces `{}`.

> Subroutines do not take any arguments nor do they return a value.

Example:

```tin
def greet {
    print "Hello, World!\n"
}
```

The `retrun` statement can be used to exit a subroutine:
```
def earlyExit {
    print "Before return\n"
    return
    print "After return: unreachable!\n"
}

call earlyExit
```
output:
```
Before return
```

### Calling Subroutines

Subroutines are called using the `call` keyword followed by the subroutine name.

Example:

```tin
call greet
```

## Defining and Using Variables

### Defining Variables

Variables in Tin are defined by assigning a value to a name using the `=` operator. Variables are dynamically typed and can hold numbers or strings.

Example:

```tin
message = "Hello, World!"
count = 42
pi = 3.14
```

### Using Variables

Variables can be used in expressions and statements.

Example:

```tin
print message, "\n"
print "The count is ", count, "\n"
```

## Global Scope

Tin has a single global scope. There is no function scoping or module/file scoping. All variables and subroutines are defined in the global scope and are accessible from anywhere in the script.

## Control Flow

### If Statements

The `if` statement is used to execute a block of code conditionally. The `else` and `else if` keywords can be used for additional conditions.

Example:

```tin
if count > 10 {
    print "Count is greater than 10\n"
} else if count == 10 {
    print "Count is 10\n"
} else {
    print "Count is less than 10\n"
}
```

### While Statements

The `while` statement is used to execute a block of code repeatedly as long as a condition is true.

Example:

```tin
x = 0
while x < 5 {
    print "x is ", x, "\n"
    x = x + 1
}
```

## Data Types

### Numbers

Numbers in Tin can be of type `int` or `float64`. They are used in mathematical expressions and comparisons.

Example:

```tin
a = 10
b = 3.14
result = a * b
```

### Strings

Strings in Tin are sequences of characters enclosed in double quotes `"`. Strings can be concatenated using the `+` operator.

Example:

```tin
greeting = "Hello, " + "World!"
print greeting, "\n"
```

## Falsy Values

In Tin, the following values are considered falsy:
- The number `0`
- The empty string `""`

Any other value is considered truthy.

### Example of Falsy Values

```tin
if 0 {
    print "This will not print\n"
} else {
    print "0 is falsy\n"
}

if "" {
    print "This will not print\n"
} else {
    print "Empty string is falsy\n"
}
```

## Builtin Functions

### print
- **Syntax**: `print <arg1>, <arg2>, ...`
- **Description**: Prints the given arguments to the standard out
- **Example**: `print: "Hello, World times ", 10, \n"`

### println
- **Syntax**: `println <arg1>, <arg2>, ...`
- **Description**: Prints the given arguments to the standard out and adds a newline character at the end
- **Example**: `print: "Hello, World times ", 10"`

### wait
- **Syntax**: `wait <milliseconds>`
- **Description**: Waits the given amout of milliseconds
- **Example**: `wait 2000`

## Editor Plugins
In the `editor` directory you will find plugins for different editors. Currently for (help is welcome):
 - [VS Code](https://code.visualstudio.com/)

 The `readme.md` in each directory explains how to install them.