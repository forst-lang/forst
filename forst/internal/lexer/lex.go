// Package lexer converts Forst code into tokens.
package lexer

import (
	"bufio"
	"bytes"
	"unicode"

	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
)

// Lex converts the input into a slice of tokens
func (l *Lexer) Lex() []ast.Token {
	reader := bufio.NewReader(bytes.NewReader(l.input))
	l.location = LexerLocation{
		Path: l.context.FilePath,
	}

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			break
		}
		l.location.LineNum++
		line = bytes.TrimRight(line, "\n")
		l.location.Column = 0

		l.log.Tracef("Processing line %d: %q\n", l.location.LineNum, line)

		// Handle accumulating block comment
		if l.accumulatingBlockComment {
			l.blockCommentValue += "\n"
			for l.location.Column < len(line) {
				if l.location.Column+1 < len(line) && line[l.location.Column] == '/' && line[l.location.Column+1] == '*' {
					l.blockCommentValue += "/*"
					l.blockCommentNesting++
					l.location.Column += 2
					continue
				}
				if l.location.Column+1 < len(line) && line[l.location.Column] == '*' && line[l.location.Column+1] == '/' {
					l.blockCommentValue += "*/"
					l.blockCommentNesting--
					l.location.Column += 2
					if l.blockCommentNesting == 0 {
						// Emit the full block comment token
						l.storeToken(ast.Token{
							Type:   ast.TokenComment,
							Value:  l.blockCommentValue,
							Path:   l.location.Path,
							Line:   l.blockCommentStartLine,
							Column: l.blockCommentStartCol,
						})
						l.accumulatingBlockComment = false
						l.blockCommentValue = ""
						break
					}
					continue
				}
				l.blockCommentValue += string(line[l.location.Column])
				l.location.Column++
			}
			if l.accumulatingBlockComment {
				continue
			}
		}

		// Handle accumulating backtick string
		if l.accumulatingBacktickString {
			l.backtickStringValue += "\n"
			lineStart := 0
			if l.location.Column > 0 {
				lineStart = l.location.Column
			}
			lineRest := line[lineStart:]
			endIdx := bytes.IndexByte(lineRest, '`')
			if endIdx != -1 {
				// Found closing backtick
				l.backtickStringValue += string(lineRest[:endIdx])
				// Include backticks in the value
				valueWithBackticks := "`" + l.backtickStringValue + "`"
				l.storeToken(ast.Token{
					Type:   ast.TokenStringLiteral,
					Value:  valueWithBackticks,
					Path:   l.location.Path,
					Line:   l.backtickStringStartLine,
					Column: l.backtickStringStartCol,
				})
				l.accumulatingBacktickString = false
				l.backtickStringValue = ""
				l.location.Column = lineStart + endIdx + 1
				// Continue lexing after the closing backtick
			} else {
				l.backtickStringValue += string(lineRest)
				continue
			}
		}

		for len(line[l.location.Column:]) > 0 {
			for l.location.Column < len(line) && unicode.IsSpace(rune(line[l.location.Column])) {
				l.location.Column++
			}
			if l.location.Column >= len(line) {
				break
			}

			// Handle string literals (double quotes, single quotes)
			if line[l.location.Column] == '"' || line[l.location.Column] == '\'' {
				token, newCol := processStringLiteral(line, l.location.Column, l.location.Path, l.location.LineNum)
				l.storeToken(token)
				l.location.Column = newCol
				continue
			}

			// Handle start of backtick string
			if line[l.location.Column] == '`' {
				l.accumulatingBacktickString = true
				l.backtickStringValue = ""
				l.backtickStringStartLine = l.location.LineNum
				l.backtickStringStartCol = l.location.Column + 1
				// Check if closing backtick is on the same line
				lineRest := line[l.location.Column+1:]
				endIdx := bytes.IndexByte(lineRest, '`')
				if endIdx != -1 {
					l.backtickStringValue = string(lineRest[:endIdx])
					// Include backticks in the value
					valueWithBackticks := "`" + l.backtickStringValue + "`"
					l.storeToken(ast.Token{
						Type:   ast.TokenStringLiteral,
						Value:  valueWithBackticks,
						Path:   l.location.Path,
						Line:   l.backtickStringStartLine,
						Column: l.backtickStringStartCol,
					})
					l.accumulatingBacktickString = false
					l.backtickStringValue = ""
					l.location.Column += endIdx + 2 // skip both backticks
					continue
				} else {
					l.backtickStringValue = string(line[l.location.Column+1:])
					// Will continue accumulating in the next line
					break
				}
			}

			// Start of block comment
			if l.location.Column+1 < len(line) && line[l.location.Column] == '/' && line[l.location.Column+1] == '*' {
				l.accumulatingBlockComment = true
				l.blockCommentValue = "/*"
				l.blockCommentStartLine = l.location.LineNum
				l.blockCommentStartCol = l.location.Column + 1
				l.blockCommentNesting = 1
				l.location.Column += 2
				for l.location.Column < len(line) {
					if l.location.Column+1 < len(line) && line[l.location.Column] == '/' && line[l.location.Column+1] == '*' {
						l.blockCommentValue += "/*"
						l.blockCommentNesting++
						l.location.Column += 2
						continue
					}
					if l.location.Column+1 < len(line) && line[l.location.Column] == '*' && line[l.location.Column+1] == '/' {
						l.blockCommentValue += "*/"
						l.blockCommentNesting--
						l.location.Column += 2
						if l.blockCommentNesting == 0 {
							l.storeToken(ast.Token{
								Type:   ast.TokenComment,
								Value:  l.blockCommentValue,
								Path:   l.location.Path,
								Line:   l.blockCommentStartLine,
								Column: l.blockCommentStartCol,
							})
							l.accumulatingBlockComment = false
							l.blockCommentValue = ""
							break
						}
						continue
					}
					l.blockCommentValue += string(line[l.location.Column])
					l.location.Column++
				}
				if l.accumulatingBlockComment {
					// Continue accumulating in the next line
					break
				}
				continue
			}

			// Check for special characters that should be separate tokens
			if isSpecialChar(line[l.location.Column]) {
				token, newColumn := processSpecialChar(line, l.location.Column, l.location.Path, l.location.LineNum)
				l.storeToken(token)
				l.location.Column = newColumn

				if token.Type == ast.TokenComment {
					break
				}

				continue
			}

			// Otherwise, process as a word/identifier/number
			token, newCol := processWord(line, l.location.Column, l.location.Path, l.location.LineNum)
			l.storeToken(token)
			l.location.Column = newCol
		}
	}

	// Emit EOF token
	l.storeToken(ast.Token{
		Type:   ast.TokenEOF,
		Value:  "",
		Path:   l.location.Path,
		Line:   l.location.LineNum + 1,
		Column: 1,
	})

	return l.tokens
}

// storeToken adds a token to the lexer's token list
func (l *Lexer) storeToken(token ast.Token) {
	log.Tracef("Storing token: Type=%q, Value=%q, Line=%d, Column=%d\n", token.Type, token.Value, token.Line, token.Column)
	l.tokens = append(l.tokens, token)
}
