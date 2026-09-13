package san

import "testing"

// emptyBoard is a bare board for tests that want to place a handful of
// pieces by hand rather than start from the full opening position.
func emptyBoard(toMove Color) *Board {
	return &Board{ToMove: toMove}
}

func mustParse(t *testing.T, s string) *Move {
	t.Helper()
	mv, err := Parse(s, false)
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", s, err)
	}
	return mv
}

func TestNewBoardStartingPosition(t *testing.T) {
	b := NewBoard()

	cases := []struct {
		file, rank byte
		wantPiece  Piece
		wantColor  Color
	}{
		{'a', '1', Rook, White},
		{'e', '1', King, White},
		{'h', '1', Rook, White},
		{'e', '2', Pawn, White},
		{'e', '7', Pawn, Black},
		{'d', '8', Queen, Black},
		{'g', '8', Knight, Black},
	}
	for _, tc := range cases {
		piece, color, ok := b.PieceAt(tc.file, tc.rank)
		if !ok || piece != tc.wantPiece || color != tc.wantColor {
			t.Errorf("PieceAt(%c,%c) = (%c, %c, %v), want (%c, %c, true)",
				tc.file, tc.rank, piece, color, ok, tc.wantPiece, tc.wantColor)
		}
	}

	for file := byte('a'); file <= 'h'; file++ {
		for rank := byte('3'); rank <= '6'; rank++ {
			if _, _, ok := b.PieceAt(file, rank); ok {
				t.Errorf("PieceAt(%c,%c) occupied on an empty rank", file, rank)
			}
		}
	}
	if b.ToMove != White {
		t.Errorf("ToMove = %c, want White", b.ToMove)
	}
}

func TestApplyOpeningSequence(t *testing.T) {
	b := NewBoard()
	moves := []string{"e4", "e5", "Nf3", "Nc6", "Bb5"}
	for _, s := range moves {
		if err := b.Apply(mustParse(t, s)); err != nil {
			t.Fatalf("Apply(%q) returned error: %v", s, err)
		}
	}

	if _, _, ok := b.PieceAt('e', '2'); ok {
		t.Error("e2 should be empty after e4")
	}
	if piece, color, ok := b.PieceAt('e', '4'); !ok || piece != Pawn || color != White {
		t.Errorf("e4 should hold a white pawn, got (%c, %c, %v)", piece, color, ok)
	}
	if piece, color, ok := b.PieceAt('f', '3'); !ok || piece != Knight || color != White {
		t.Errorf("f3 should hold a white knight, got (%c, %c, %v)", piece, color, ok)
	}
	if piece, color, ok := b.PieceAt('b', '5'); !ok || piece != Bishop || color != White {
		t.Errorf("b5 should hold a white bishop, got (%c, %c, %v)", piece, color, ok)
	}
	if b.ToMove != Black {
		t.Errorf("ToMove = %c, want Black after an odd number of half-moves", b.ToMove)
	}
}

func TestApplyRejectsPawnPushOntoOccupiedSquare(t *testing.T) {
	b := NewBoard()
	for _, s := range []string{"e4", "e5"} {
		if err := b.Apply(mustParse(t, s)); err != nil {
			t.Fatalf("Apply(%q) returned error: %v", s, err)
		}
	}
	if err := b.Apply(mustParse(t, "e5")); err == nil {
		t.Fatal("Apply(e5) onto an occupied square with no capture flag succeeded, want error")
	}
}

func TestApplyRejectsCaptureWithNothingToCapture(t *testing.T) {
	b := NewBoard()
	if err := b.Apply(mustParse(t, "Nxf3")); err == nil {
		t.Fatal("Apply(Nxf3) with an empty destination succeeded, want error")
	}
}

func TestApplyRejectsMoveWithNoReachablePiece(t *testing.T) {
	b := emptyBoard(White)
	b.squares[fileIndex('a')][rankIndex('1')] = square{occupied: true, kind: Rook, color: White}
	b.squares[fileIndex('a')][rankIndex('4')] = square{occupied: true, kind: Pawn, color: White}

	if err := b.Apply(mustParse(t, "Ra8")); err == nil {
		t.Fatal("Apply(Ra8) through a blocked path succeeded, want error")
	}
}

func TestApplyDisambiguatesAmbiguousMove(t *testing.T) {
	b := emptyBoard(White)
	b.squares[fileIndex('d')][rankIndex('5')] = square{occupied: true, kind: Knight, color: White}
	b.squares[fileIndex('f')][rankIndex('5')] = square{occupied: true, kind: Knight, color: White}

	if err := b.Apply(mustParse(t, "Ne3")); err == nil {
		t.Fatal("Apply(Ne3) with two knights able to reach e3 succeeded, want an ambiguous-move error")
	}

	if err := b.Apply(mustParse(t, "Nde3")); err != nil {
		t.Fatalf("Apply(Nde3) returned error: %v", err)
	}
	if _, _, ok := b.PieceAt('d', '5'); ok {
		t.Error("d5 should be empty after Nde3")
	}
	if piece, color, ok := b.PieceAt('f', '5'); !ok || piece != Knight || color != White {
		t.Error("f5 should still hold the other white knight")
	}
	if piece, color, ok := b.PieceAt('e', '3'); !ok || piece != Knight || color != White {
		t.Error("e3 should hold the moved white knight")
	}
}

func TestApplyEnPassantCapture(t *testing.T) {
	b := emptyBoard(White)
	b.squares[fileIndex('e')][rankIndex('5')] = square{occupied: true, kind: Pawn, color: White}
	b.squares[fileIndex('d')][rankIndex('5')] = square{occupied: true, kind: Pawn, color: Black}
	b.enPassantFile = 'd'

	if err := b.Apply(mustParse(t, "exd6")); err != nil {
		t.Fatalf("Apply(exd6) returned error: %v", err)
	}
	if _, _, ok := b.PieceAt('d', '5'); ok {
		t.Error("captured pawn on d5 should be gone after en passant")
	}
	if _, _, ok := b.PieceAt('e', '5'); ok {
		t.Error("e5 should be empty after the capturing pawn moves")
	}
	if piece, color, ok := b.PieceAt('d', '6'); !ok || piece != Pawn || color != White {
		t.Error("d6 should hold the capturing white pawn")
	}
}

func TestApplyRejectsEnPassantWhenUnavailable(t *testing.T) {
	b := emptyBoard(White)
	b.squares[fileIndex('e')][rankIndex('5')] = square{occupied: true, kind: Pawn, color: White}
	b.squares[fileIndex('d')][rankIndex('5')] = square{occupied: true, kind: Pawn, color: Black}
	// enPassantFile left unset: the black pawn didn't just double-move.

	if err := b.Apply(mustParse(t, "exd6")); err == nil {
		t.Fatal("Apply(exd6) succeeded without an open en passant capture, want error")
	}
}

func TestApplyPromotion(t *testing.T) {
	b := emptyBoard(White)
	b.squares[fileIndex('e')][rankIndex('7')] = square{occupied: true, kind: Pawn, color: White}

	if err := b.Apply(mustParse(t, "e8=Q")); err != nil {
		t.Fatalf("Apply(e8=Q) returned error: %v", err)
	}
	if piece, color, ok := b.PieceAt('e', '8'); !ok || piece != Queen || color != White {
		t.Errorf("e8 should hold a white queen, got (%c, %c, %v)", piece, color, ok)
	}
}

func TestApplyCastleKingside(t *testing.T) {
	b := emptyBoard(White)
	b.squares[fileIndex('e')][rankIndex('1')] = square{occupied: true, kind: King, color: White}
	b.squares[fileIndex('h')][rankIndex('1')] = square{occupied: true, kind: Rook, color: White}
	b.whiteKingside = true

	if err := b.Apply(mustParse(t, "O-O")); err != nil {
		t.Fatalf("Apply(O-O) returned error: %v", err)
	}
	if piece, color, ok := b.PieceAt('g', '1'); !ok || piece != King || color != White {
		t.Error("g1 should hold the white king after O-O")
	}
	if piece, color, ok := b.PieceAt('f', '1'); !ok || piece != Rook || color != White {
		t.Error("f1 should hold the white rook after O-O")
	}
	if b.whiteKingside || b.whiteQueenside {
		t.Error("castling rights should be cleared after castling")
	}
}

func TestApplyCastleRejectsBlockedPath(t *testing.T) {
	b := emptyBoard(White)
	b.squares[fileIndex('e')][rankIndex('1')] = square{occupied: true, kind: King, color: White}
	b.squares[fileIndex('f')][rankIndex('1')] = square{occupied: true, kind: Bishop, color: White}
	b.squares[fileIndex('h')][rankIndex('1')] = square{occupied: true, kind: Rook, color: White}
	b.whiteKingside = true

	if err := b.Apply(mustParse(t, "O-O")); err == nil {
		t.Fatal("Apply(O-O) through a blocked path succeeded, want error")
	}
}

func TestApplyCastleRejectsWithoutRights(t *testing.T) {
	b := emptyBoard(White)
	b.squares[fileIndex('e')][rankIndex('1')] = square{occupied: true, kind: King, color: White}
	b.squares[fileIndex('h')][rankIndex('1')] = square{occupied: true, kind: Rook, color: White}
	// whiteKingside left false, as if the king had already moved.

	if err := b.Apply(mustParse(t, "O-O")); err == nil {
		t.Fatal("Apply(O-O) without castling rights succeeded, want error")
	}
}
