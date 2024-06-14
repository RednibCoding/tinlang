package tinvm

import "fmt"

// #################################################################
//
//	Custom Functions
//
// #################################################################
func customPrintFunction(args []interface{}) error {
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
