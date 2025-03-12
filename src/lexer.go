package main

import (
	"strings"
	"unicode"
)

// Lexer: Converts Forst code into tokens
func Lexer(input string) []Token {
	tokens := []Token{}
	words := strings.Fields(input)

	for _, word := range words {
		switch word {
		case "fn":
			tokens = append(tokens, Token{TokenFunc, word})
		case "->":
			tokens = append(tokens, Token{TokenArrow, word})
		case "(":
			tokens = append(tokens, Token{TokenLParen, word})
		case ")":
			tokens = append(tokens, Token{TokenRParen, word})
		case "{":
			tokens = append(tokens, Token{TokenLBrace, word})
		case "}":
			tokens = append(tokens, Token{TokenRBrace, word})
		case "return":
			tokens = append(tokens, Token{TokenReturn, word})
		case "assert":
			tokens = append(tokens, Token{TokenAssert, word})
		case "or":
			tokens = append(tokens, Token{TokenOr, word})
		default:
			if unicode.IsDigit(rune(word[0])) {
				tokens = append(tokens, Token{TokenInt, word})
			} else {
				tokens = append(tokens, Token{TokenIdent, word})
			}
		}
	}

	tokens = append(tokens, Token{TokenEOF, ""}) // End of file token
	return tokens
}