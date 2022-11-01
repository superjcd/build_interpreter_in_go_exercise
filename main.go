package main

import (
	"fmt"
	"monkey/lexer"
)

func main() {
	input := `=+(){},;`
	lex := lexer.New(input)
	tok1 := lex.NextToken()
	fmt.Printf("Token %#v", tok1)

	tok2 := lex.NextToken()
	fmt.Printf("Token %#v", tok2)
}
