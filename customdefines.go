package tinvm

import (
	"fmt"
	"time"
)

// #################################################################
//
//	Custom Functions
//
// #################################################################
func customFunc_Print(vm *TinVM, args []interface{}) error {
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
	return nil
}

func customFunc_Println(vm *TinVM, args []interface{}) error {
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			fmt.Println(v)
		case int:
			fmt.Println(v)
		case float64:
			fmt.Println(v)
		default:
			return fmt.Errorf("unsupported argument type")
		}
	}
	return nil
}

func customFunction_Wait(vm *TinVM, args []interface{}) error {
	if len(args) != 1 {
		return fmt.Errorf("wait requires exactly 1 argument")
	}

	// Using type assertions to check if x and y are of type int
	ms, ok := args[0].(int)

	if !ok {
		return fmt.Errorf("argument must be of type int, got: %T", ms)
	}

	time.Sleep(time.Duration(ms) * time.Millisecond)

	return nil
}
