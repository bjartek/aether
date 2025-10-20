package ui

import "fmt"

// Helper to print actual output for test verification
func PrintWrappedHex() {
	hexString := "0x61636365737328616c6c2920636f6e74726163742046434c207b0a202061636365737328616c6c29206c65742073746f72616765506174683a2053746f72616765506174680a0a202061636365737328616c6c29207374727563742046434c4b6579207b0a20202020616363"
	
	fmt.Println("=== 80 char width with 4-space indent ===")
	result80 := FormatFieldValueWithRegistry(hexString, "    ", nil, false, 80)
	fmt.Printf("%q\n\n", result80)
	
	fmt.Println("=== 50 char width with 2-space indent ===")
	result50 := FormatFieldValueWithRegistry(hexString, "  ", nil, false, 50)
	fmt.Printf("%q\n\n", result50)
}
