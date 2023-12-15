package token

// Position defines the position in the input stream.
type Position struct {
	Line   int
	Column int
}

// NextLine advances the position to the next line and column 0.
func (p *Position) NextLine() {
	p.Line++
	p.Column = 0
}
