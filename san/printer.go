package san

import "strings"

// String renders the move in canonical strict SAN, regardless of how (or
// how leniently) it was parsed: capital piece letters, 'O' for castling, an
// explicit 'x' for every capture, and '=' before any promotion piece.
func (m *Move) String() string {
	if m.CastleKingside {
		return "O-O" + m.suffix()
	}
	if m.CastleQueenside {
		return "O-O-O" + m.suffix()
	}

	var b strings.Builder
	if m.Piece != Pawn {
		b.WriteByte(byte(m.Piece))
	}
	if m.FromFile != 0 {
		b.WriteByte(m.FromFile)
	}
	if m.FromRank != 0 {
		b.WriteByte(m.FromRank)
	}
	if m.Capture {
		b.WriteByte('x')
	}
	b.WriteByte(m.DestFile)
	b.WriteByte(m.DestRank)
	if m.Promotion != Pawn {
		b.WriteByte('=')
		b.WriteByte(byte(m.Promotion))
	}
	b.WriteString(m.suffix())
	return b.String()
}

func (m *Move) suffix() string {
	switch {
	case m.Mate:
		return "#"
	case m.Check:
		return "+"
	default:
		return ""
	}
}
