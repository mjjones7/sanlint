package san

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ParseError describes why a single SAN token failed to parse.
type ParseError struct {
	Input  string
	Reason string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("san: invalid move %q: %s", e.Input, e.Reason)
}

var (
	castleRe = regexp.MustCompile(`^(O-O-O|O-O)([+#]?)$`)
	moveRe   = regexp.MustCompile(`^([NBRQK]?)([a-h]?)([1-8]?)(x?)([a-h])([1-8])(=[NBRQ])?([+#]?)$`)
)

// Parse validates a single SAN move token and returns its structured form.
//
// In strict mode (lenient=false), the input must already be canonical SAN:
// capital piece letters, an 'x' marking every capture, 'O' (the letter, not
// the digit zero) for castling, and '=' preceding a promotion piece. Any
// deviation is rejected.
//
// In lenient mode, the input is first run through Normalize, which rewrites
// a fixed set of common non-canonical spellings into their strict
// equivalents before parsing proceeds. See Normalize for the exact list.
func Parse(s string, lenient bool) (*Move, error) {
	raw := s
	if lenient {
		s = Normalize(s)
	}
	if s == "" {
		return nil, &ParseError{raw, "empty move"}
	}

	if m := castleRe.FindStringSubmatch(s); m != nil {
		mv := &Move{
			CastleKingside:  m[1] == "O-O",
			CastleQueenside: m[1] == "O-O-O",
			Check:           m[2] == "+",
			Mate:            m[2] == "#",
		}
		return mv, nil
	}

	m := moveRe.FindStringSubmatch(s)
	if m == nil {
		return nil, &ParseError{raw, "does not match SAN move syntax"}
	}
	pieceStr, fromFileStr, fromRankStr := m[1], m[2], m[3]
	captureStr, destFileStr, destRankStr := m[4], m[5], m[6]
	promoStr, suffixStr := m[7], m[8]

	mv := &Move{
		Piece:    Pawn,
		Capture:  captureStr == "x",
		DestFile: destFileStr[0],
		DestRank: destRankStr[0],
		Check:    suffixStr == "+",
		Mate:     suffixStr == "#",
	}
	if pieceStr != "" {
		mv.Piece = Piece(pieceStr[0])
	}
	if fromFileStr != "" {
		mv.FromFile = fromFileStr[0]
	}
	if fromRankStr != "" {
		mv.FromRank = fromRankStr[0]
	}
	if promoStr != "" {
		mv.Promotion = Piece(promoStr[1])
	}

	if err := validate(mv); err != nil {
		return nil, &ParseError{raw, err.Error()}
	}
	return mv, nil
}

func validate(m *Move) error {
	if m.Piece == Pawn {
		if m.FromRank != 0 {
			return errors.New("pawn moves cannot have rank disambiguation")
		}
		if m.Capture && m.FromFile == 0 {
			return errors.New("pawn captures must specify the origin file, e.g. exd5")
		}
		if !m.Capture && m.FromFile != 0 {
			return errors.New("non-capturing pawn moves cannot have a file prefix")
		}
		onLastRank := m.DestRank == '1' || m.DestRank == '8'
		if onLastRank && m.Promotion == Pawn {
			return errors.New("pawn move to the last rank must include a promotion, e.g. =Q")
		}
		if !onLastRank && m.Promotion != Pawn {
			return errors.New("promotion is only legal on the last rank")
		}
		return nil
	}

	if m.Promotion != Pawn {
		return errors.New("only pawns can promote")
	}
	if m.Piece == King && (m.FromFile != 0 || m.FromRank != 0) {
		return errors.New("king moves cannot be disambiguated, there is only one king")
	}
	return nil
}

var (
	epSuffixRe     = regexp.MustCompile(`\s*e\.p\.$`)
	annotationRe   = regexp.MustCompile(`[!?]+$`)
	castleDigitRe  = regexp.MustCompile(`(?i)^(0-0-0|0-0|o-o-o|o-o)([+#]?)$`)
	missingPromoRe = regexp.MustCompile(`^([a-h]x)?([a-h][18])([NBRQ])([+#]?)$`)
)

// Normalize rewrites a fixed set of common non-canonical SAN spellings into
// the strict form Parse expects. It is applied automatically when Parse is
// called with lenient=true; it never guesses at anything that depends on
// board state (e.g. it will not insert a missing capture 'x').
//
// Recognized relaxations:
//   - digit castling: "0-0", "0-0-0" (any case) -> "O-O", "O-O-O"
//   - a missing '=' before a promotion piece: "e8Q" -> "e8=Q"
//   - trailing annotation glyphs: "Nf3!?", "Qxb7??" -> glyphs stripped
//   - a trailing en passant marker: "exd6 e.p." -> "exd6"
//   - leading/trailing whitespace
func Normalize(s string) string {
	s = strings.TrimSpace(s)
	s = epSuffixRe.ReplaceAllString(s, "")
	s = annotationRe.ReplaceAllString(s, "")

	if m := castleDigitRe.FindStringSubmatch(s); m != nil {
		body := "O-O"
		if strings.Count(m[1], "-") == 2 {
			body = "O-O-O"
		}
		return body + m[2]
	}

	if m := missingPromoRe.FindStringSubmatch(s); m != nil {
		return m[1] + m[2] + "=" + m[3] + m[4]
	}

	return s
}
