package san

import "fmt"

// Color identifies a side to move, or the owner of a piece on a Board.
type Color byte

const (
	White Color = 'w'
	Black Color = 'b'
)

func (c Color) other() Color {
	if c == White {
		return Black
	}
	return White
}

func sideName(c Color) string {
	if c == White {
		return "white"
	}
	return "black"
}

// square is a single board square's contents. It needs its own "empty"
// state separate from Piece's zero value, since Piece(0) is Pawn (see the
// comment on Move.Promotion in move.go).
type square struct {
	occupied bool
	kind     Piece
	color    Color
}

// Board tracks piece placement well enough to tell whether a parsed Move
// actually corresponds to a piece that can make it: the right kind of
// piece sits on a square that can reach the destination, sliding pieces
// have a clear path, and the capture flag matches what's really on the
// destination square (including en passant).
//
// It does not compute attacked squares, so it will not stop a king from
// moving into check, and it trusts the Check/Mate flags on Move rather
// than deriving them itself.
type Board struct {
	squares [8][8]square // squares[file][rank], both 0-indexed: a-h, 1-8
	ToMove  Color

	// enPassantFile is the file of a pawn that just advanced two squares,
	// or 0 if the previous move didn't open an en passant capture.
	enPassantFile byte

	whiteKingside, whiteQueenside bool
	blackKingside, blackQueenside bool
}

// NewBoard returns a Board set up in the standard starting position, White
// to move, with full castling rights for both sides.
func NewBoard() *Board {
	b := &Board{
		ToMove:         White,
		whiteKingside:  true,
		whiteQueenside: true,
		blackKingside:  true,
		blackQueenside: true,
	}

	backRank := [8]Piece{Rook, Knight, Bishop, Queen, King, Bishop, Knight, Rook}
	for f := 0; f < 8; f++ {
		b.squares[f][0] = square{occupied: true, kind: backRank[f], color: White}
		b.squares[f][1] = square{occupied: true, kind: Pawn, color: White}
		b.squares[f][6] = square{occupied: true, kind: Pawn, color: Black}
		b.squares[f][7] = square{occupied: true, kind: backRank[f], color: Black}
	}
	return b
}

// PieceAt reports what's on the given square, if anything. file and rank
// are the raw SAN bytes ('a'-'h', '1'-'8').
func (b *Board) PieceAt(file, rank byte) (piece Piece, color Color, occupied bool) {
	sq := b.squares[fileIndex(file)][rankIndex(rank)]
	return sq.kind, sq.color, sq.occupied
}

// Apply plays mv against the board for the side currently on move. On
// success it updates piece placement, castling rights, and the en passant
// file, then flips ToMove. On failure the board is left unchanged.
func (b *Board) Apply(mv *Move) error {
	side := b.ToMove

	var err error
	if mv.CastleKingside || mv.CastleQueenside {
		err = b.applyCastle(mv, side)
	} else {
		err = b.applyMove(mv, side)
	}
	if err != nil {
		return err
	}
	b.ToMove = side.other()
	return nil
}

func (b *Board) applyMove(mv *Move, side Color) error {
	destFile := fileIndex(mv.DestFile)
	destRank := rankIndex(mv.DestRank)
	dest := b.squares[destFile][destRank]

	capturedEnPassant, err := b.checkCapture(mv, side, dest)
	if err != nil {
		return err
	}

	fromFile, fromRank, err := b.locateOrigin(mv, side, destFile, destRank)
	if err != nil {
		return err
	}

	moving := b.squares[fromFile][fromRank]

	if capturedEnPassant {
		// The captured pawn sits beside the capturing pawn's start
		// square, not on the (empty) destination square.
		b.squares[destFile][fromRank] = square{}
	}

	b.squares[fromFile][fromRank] = square{}
	if mv.Promotion != Pawn {
		moving.kind = mv.Promotion
	}
	b.squares[destFile][destRank] = moving

	switch {
	case moving.kind == King:
		b.clearCastleRights(side)
	case moving.kind == Rook:
		b.clearRookRight(side, fromFile, fromRank)
	}
	if dest.occupied && dest.kind == Rook {
		b.clearRookRight(side.other(), destFile, destRank)
	}

	if mv.Piece == Pawn && absInt(destRank-fromRank) == 2 {
		b.enPassantFile = mv.DestFile
	} else {
		b.enPassantFile = 0
	}
	return nil
}

// checkCapture validates mv's capture flag against what's actually on the
// destination square and reports whether this is an en passant capture.
func (b *Board) checkCapture(mv *Move, side Color, dest square) (enPassant bool, err error) {
	if mv.Piece != Pawn {
		if mv.Capture {
			if !dest.occupied || dest.color == side {
				return false, fmt.Errorf("no %s piece on %c%c to capture", sideName(side.other()), mv.DestFile, mv.DestRank)
			}
			return false, nil
		}
		if dest.occupied {
			return false, fmt.Errorf("%c%c is occupied", mv.DestFile, mv.DestRank)
		}
		return false, nil
	}

	if !mv.Capture {
		if dest.occupied {
			return false, fmt.Errorf("%c%c is occupied", mv.DestFile, mv.DestRank)
		}
		return false, nil
	}
	if dest.occupied {
		if dest.color == side {
			return false, fmt.Errorf("no %s piece on %c%c to capture", sideName(side.other()), mv.DestFile, mv.DestRank)
		}
		return false, nil
	}

	epRank := byte('6')
	if side == Black {
		epRank = '3'
	}
	if mv.DestRank != epRank || b.enPassantFile != mv.DestFile {
		return false, fmt.Errorf("no piece on %c%c to capture and no en passant available", mv.DestFile, mv.DestRank)
	}
	return true, nil
}

// locateOrigin finds the single square holding a piece of mv.Piece's kind
// and the side to move that can legally reach the destination, honoring
// any disambiguation in mv.FromFile/mv.FromRank.
func (b *Board) locateOrigin(mv *Move, side Color, destFile, destRank int) (file, rank int, err error) {
	found := false
	for f := 0; f < 8; f++ {
		for r := 0; r < 8; r++ {
			sq := b.squares[f][r]
			if !sq.occupied || sq.kind != mv.Piece || sq.color != side {
				continue
			}
			if mv.FromFile != 0 && f != fileIndex(mv.FromFile) {
				continue
			}
			if mv.FromRank != 0 && r != rankIndex(mv.FromRank) {
				continue
			}
			if !b.pieceReaches(mv.Piece, side, f, r, destFile, destRank, mv.Capture) {
				continue
			}
			if found {
				return 0, 0, fmt.Errorf("ambiguous move: more than one %s %s can reach %c%c",
					sideName(side), pieceName(mv.Piece), mv.DestFile, mv.DestRank)
			}
			file, rank, found = f, r, true
		}
	}
	if !found {
		return 0, 0, fmt.Errorf("no %s %s can reach %c%c", sideName(side), pieceName(mv.Piece), mv.DestFile, mv.DestRank)
	}
	return file, rank, nil
}

func (b *Board) pieceReaches(kind Piece, side Color, fromFile, fromRank, toFile, toRank int, capture bool) bool {
	df := toFile - fromFile
	dr := toRank - fromRank

	switch kind {
	case Pawn:
		return b.pawnReaches(side, fromFile, fromRank, toFile, toRank, capture)
	case Knight:
		ad, ar := absInt(df), absInt(dr)
		return (ad == 1 && ar == 2) || (ad == 2 && ar == 1)
	case Bishop:
		return absInt(df) == absInt(dr) && df != 0 && b.pathClear(fromFile, fromRank, toFile, toRank)
	case Rook:
		return (df == 0) != (dr == 0) && b.pathClear(fromFile, fromRank, toFile, toRank)
	case Queen:
		straight := (df == 0) != (dr == 0)
		diagonal := absInt(df) == absInt(dr) && df != 0
		return (straight || diagonal) && b.pathClear(fromFile, fromRank, toFile, toRank)
	case King:
		return absInt(df) <= 1 && absInt(dr) <= 1 && (df != 0 || dr != 0)
	}
	return false
}

func (b *Board) pawnReaches(side Color, fromFile, fromRank, toFile, toRank int, capture bool) bool {
	dir, startRank := 1, 1
	if side == Black {
		dir, startRank = -1, 6
	}

	if capture {
		return absInt(toFile-fromFile) == 1 && toRank-fromRank == dir
	}
	if toFile != fromFile {
		return false
	}
	if toRank-fromRank == dir {
		return true
	}
	if fromRank == startRank && toRank-fromRank == 2*dir {
		return !b.squares[fromFile][fromRank+dir].occupied
	}
	return false
}

// pathClear reports whether every square strictly between from and to is
// empty. It assumes the two squares lie on a shared rank, file, or
// diagonal, which callers must have already checked.
func (b *Board) pathClear(fromFile, fromRank, toFile, toRank int) bool {
	df, dr := sign(toFile-fromFile), sign(toRank-fromRank)
	f, r := fromFile+df, fromRank+dr
	for f != toFile || r != toRank {
		if b.squares[f][r].occupied {
			return false
		}
		f += df
		r += dr
	}
	return true
}

func (b *Board) applyCastle(mv *Move, side Color) error {
	rank := 0
	if side == Black {
		rank = 7
	}

	have := b.kingsideRight(side)
	if mv.CastleQueenside {
		have = b.queensideRight(side)
	}
	if !have {
		return fmt.Errorf("%s has lost the right to castle %s", sideName(side), castleName(mv))
	}

	kingFile := fileIndex('e')
	rookFile, newKingFile, newRookFile := fileIndex('h'), fileIndex('g'), fileIndex('f')
	between := []int{fileIndex('f'), fileIndex('g')}
	if mv.CastleQueenside {
		rookFile, newKingFile, newRookFile = fileIndex('a'), fileIndex('c'), fileIndex('d')
		between = []int{fileIndex('b'), fileIndex('c'), fileIndex('d')}
	}

	king := b.squares[kingFile][rank]
	rook := b.squares[rookFile][rank]
	if !king.occupied || king.kind != King || king.color != side {
		return fmt.Errorf("no %s king in place to castle", sideName(side))
	}
	if !rook.occupied || rook.kind != Rook || rook.color != side {
		return fmt.Errorf("no %s rook in place to castle %s", sideName(side), castleName(mv))
	}
	for _, f := range between {
		if b.squares[f][rank].occupied {
			return fmt.Errorf("castling path is blocked")
		}
	}

	b.squares[kingFile][rank] = square{}
	b.squares[rookFile][rank] = square{}
	b.squares[newKingFile][rank] = king
	b.squares[newRookFile][rank] = rook

	b.clearCastleRights(side)
	b.enPassantFile = 0
	return nil
}

func (b *Board) kingsideRight(side Color) bool {
	if side == White {
		return b.whiteKingside
	}
	return b.blackKingside
}

func (b *Board) queensideRight(side Color) bool {
	if side == White {
		return b.whiteQueenside
	}
	return b.blackQueenside
}

func (b *Board) clearCastleRights(side Color) {
	if side == White {
		b.whiteKingside, b.whiteQueenside = false, false
	} else {
		b.blackKingside, b.blackQueenside = false, false
	}
}

// clearRookRight drops castling rights on the side of the board where a
// rook just moved away from or was captured on its home square.
func (b *Board) clearRookRight(side Color, file, rank int) {
	homeRank := 0
	if side == Black {
		homeRank = 7
	}
	if rank != homeRank {
		return
	}
	switch file {
	case fileIndex('a'):
		if side == White {
			b.whiteQueenside = false
		} else {
			b.blackQueenside = false
		}
	case fileIndex('h'):
		if side == White {
			b.whiteKingside = false
		} else {
			b.blackKingside = false
		}
	}
}

func castleName(mv *Move) string {
	if mv.CastleKingside {
		return "kingside"
	}
	return "queenside"
}

func pieceName(p Piece) string {
	switch p {
	case Pawn:
		return "pawn"
	case Knight:
		return "knight"
	case Bishop:
		return "bishop"
	case Rook:
		return "rook"
	case Queen:
		return "queen"
	case King:
		return "king"
	default:
		return "piece"
	}
}

func fileIndex(f byte) int { return int(f - 'a') }
func rankIndex(r byte) int { return int(r - '1') }

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}
