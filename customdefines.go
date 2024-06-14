package tinvm

import "fmt"

// #################################################################
//
//	Custom Functions
//
// #################################################################
func customFunc_Print(args []interface{}) error {
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

func customFunc_Println(args []interface{}) error {
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
