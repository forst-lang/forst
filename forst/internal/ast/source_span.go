package ast

import "unicode/utf8"

// SourceSpan is a half-open range in source: [StartLine:StartCol, EndLine:EndCol)
// Line and column are 1-based (matching lexer tokens). EndCol is exclusive on EndLine.
type SourceSpan struct {
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

// IsSet reports whether the span was populated by the parser.
func (s SourceSpan) IsSet() bool {
	return s.StartLine >= 1 && s.StartCol >= 1
}

// SpanFromToken is a span covering a single token's text.
func SpanFromToken(t Token) SourceSpan {
	w := utf8.RuneCountInString(t.Value)
	if w < 1 {
		w = 1
	}
	return SourceSpan{
		StartLine: t.Line,
		StartCol:  t.Column,
		EndLine:   t.Line,
		EndCol:    t.Column + w,
	}
}

// SpanBetweenTokens is a span from the start of start through the end of end (inclusive of end's text).
func SpanBetweenTokens(start, end Token) SourceSpan {
	w := utf8.RuneCountInString(end.Value)
	if w < 1 {
		w = 1
	}
	return SourceSpan{
		StartLine: start.Line,
		StartCol:  start.Column,
		EndLine:   end.Line,
		EndCol:    end.Column + w,
	}
}
