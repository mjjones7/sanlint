package san

import (
	"errors"
	"testing"
)

func TestParseStrictValid(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Move
	}{
		{"pawn push", "e4", Move{Piece: Pawn, DestFile: 'e', DestRank: '4'}},
		{"pawn capture", "exd5", Move{Piece: Pawn, FromFile: 'e', Capture: true, DestFile: 'd', DestRank: '5'}},
		{"knight move", "Nf3", Move{Piece: Knight, DestFile: 'f', DestRank: '3'}},
		{"knight capture", "Nxf3", Move{Piece: Knight, Capture: true, DestFile: 'f', DestRank: '3'}},
		{"file disambiguation", "Nbd7", Move{Piece: Knight, FromFile: 'b', DestFile: 'd', DestRank: '7'}},
		{"rank disambiguation", "R1a3", Move{Piece: Rook, FromRank: '1', DestFile: 'a', DestRank: '3'}},
		{"full square disambiguation", "Qh4e1", Move{Piece: Queen, FromFile: 'h', FromRank: '4', DestFile: 'e', DestRank: '1'}},
		{"promotion", "e8=Q", Move{Piece: Pawn, DestFile: 'e', DestRank: '8', Promotion: Queen}},
		{"capture promotion with check", "exd8=Q+", Move{Piece: Pawn, FromFile: 'e', Capture: true, DestFile: 'd', DestRank: '8', Promotion: Queen, Check: true}},
		{"mate suffix", "Qh5#", Move{Piece: Queen, DestFile: 'h', DestRank: '5', Mate: true}},
		{"kingside castle", "O-O", Move{CastleKingside: true}},
		{"queenside castle", "O-O-O", Move{CastleQueenside: true}},
		{"castle with check", "O-O+", Move{CastleKingside: true, Check: true}},
		{"castle with mate", "O-O-O#", Move{CastleQueenside: true, Mate: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.in, false)
			if err != nil {
				t.Fatalf("Parse(%q, false) returned error: %v", tc.in, err)
			}
			if *got != tc.want {
				t.Fatalf("Parse(%q, false) = %+v, want %+v", tc.in, *got, tc.want)
			}
		})
	}
}

func TestParseStrictRejects(t *testing.T) {
	cases := []string{
		"0-0",             // digit castling
		"e8Q",              // missing '=' before promotion
		"Nf3!?",            // trailing annotation
		"exd6 e.p.",        // en passant suffix
		"xd5",              // pawn capture missing origin file
		"ed5",              // non-capturing pawn move with file prefix
		"e4=Q",             // promotion off the last rank
		"e8",               // pawn reaching last rank without promotion
		"Ka1d2",            // king disambiguation
		"Nf3=Q",            // non-pawn promotion
		"N9d7",             // rank out of range
		"Nf9",              // rank out of range on destination
		"",                 // empty move
		"O-O-O-O",          // not a real castle
		"nf3",              // lowercase piece letter
	}

	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			mv, err := Parse(in, false)
			if err == nil {
				t.Fatalf("Parse(%q, false) = %+v, want error", in, *mv)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("Parse(%q, false) returned %T, want *ParseError", in, err)
			}
			if pe.Input != in {
				t.Fatalf("ParseError.Input = %q, want %q", pe.Input, in)
			}
		})
	}
}

func TestParseLenientAcceptsNormalizedForms(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // expected canonical String() output
	}{
		{"digit kingside castle", "0-0", "O-O"},
		{"digit queenside castle", "0-0-0", "O-O-O"},
		{"digit castle any case", "o-o", "O-O"},
		{"digit castle with check", "0-0+", "O-O+"},
		{"digit castle with annotation", "0-0!", "O-O"},
		{"missing promotion equals", "e8Q", "e8=Q"},
		{"missing promotion equals with check", "exd8Q+", "exd8=Q+"},
		{"trailing annotation glyphs", "Nf3!?", "Nf3"},
		{"trailing blunder glyph", "Qxb7??", "Qxb7"},
		{"en passant suffix", "exd6 e.p.", "exd6"},
		{"surrounding whitespace", "  Nf3  ", "Nf3"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mv, err := Parse(tc.in, true)
			if err != nil {
				t.Fatalf("Parse(%q, true) returned error: %v", tc.in, err)
			}
			if got := mv.String(); got != tc.want {
				t.Fatalf("Parse(%q, true).String() = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseLenientStillRejectsGarbage(t *testing.T) {
	cases := []string{
		"xd5",    // still missing a mandatory origin file
		"Ka1d2",  // still an illegal king disambiguation
		"blargh", // not SAN at all
	}

	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, err := Parse(in, true); err == nil {
				t.Fatalf("Parse(%q, true) succeeded, want error", in)
			}
		})
	}
}

func TestNormalizeIsIdempotentOnCanonicalInput(t *testing.T) {
	canonical := []string{"e4", "exd5", "Nf3", "O-O", "O-O-O+", "e8=Q", "Qh5#"}
	for _, s := range canonical {
		if got := Normalize(s); got != s {
			t.Errorf("Normalize(%q) = %q, want unchanged", s, got)
		}
	}
}
