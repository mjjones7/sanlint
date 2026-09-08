// Package san implements a validating parser and pretty printer for
// Standard Algebraic Notation, the notation chess games are recorded in
// ("Nf3", "exd5", "O-O", "e8=Q+").
//
// This package validates move syntax: piece letters, disambiguation shape,
// capture markers, and promotion rules. It does not know about a board
// position, so it cannot tell you whether a syntactically valid move is
// actually legal from a given position.
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
