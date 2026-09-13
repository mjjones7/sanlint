// Package san implements a validating parser and pretty printer for
// Standard Algebraic Notation, the notation chess games are recorded in
// ("Nf3", "exd5", "O-O", "e8=Q+").
//
// Parse and Move validate move syntax only: piece letters, disambiguation
// shape, capture markers, and promotion rules. A syntactically valid move
// can still be nonsense on an actual board.
//
// Board fills that gap: it tracks piece placement, applies parsed moves,
// and rejects ones that don't correspond to a real piece capable of making
// them (nothing on the source square, a blocked sliding path, a capture
// flag that doesn't match the destination square). It does not compute
// attacked squares, so it won't stop a king from moving into check.
package san

// Piece identifies the type of piece a move belongs to. The zero value,
// Pawn, also doubles as "no promotion" on Move.Promotion since a pawn can
// never be the promoted-to piece.
type Piece byte

const (
	Pawn   Piece = 0
	Knight Piece = 'N'
	Bishop Piece = 'B'
	Rook   Piece = 'R'
	Queen  Piece = 'Q'
	King   Piece = 'K'
)

// Move is a fully parsed and structurally validated SAN move.
type Move struct {
	Piece Piece

	// FromFile and FromRank hold disambiguation information: 0 if the move
	// text didn't specify it, otherwise 'a'-'h' or '1'-'8'. For pawn
	// captures, FromFile carries the mandatory origin file (e.g. the 'e'
	// in "exd5").
	FromFile byte
	FromRank byte

	Capture bool

	DestFile byte
	DestRank byte

	// Promotion is Pawn (the zero value) unless the move promotes.
	Promotion Piece

	Check bool
	Mate  bool

	CastleKingside  bool
	CastleQueenside bool
}
