# sanlint

A validating parser and pretty printer for Standard Algebraic Notation
(SAN), the notation used to record chess moves ("Nf3", "exd5", "O-O",
"e8=Q+").

PGN files and chess software disagree constantly about how loose to be with
SAN: some write castling as `0-0` instead of `O-O`, some drop the `=` before
a promotion piece, some leave `!?` annotations glued onto the move itself.
Feeding that straight into something expecting strict SAN breaks. sanlint's
job is to draw a hard line by default and only cross it when you explicitly
ask it to.

## Status

Early. `san.Parse` validates move *syntax* — legal formation,
disambiguation shape, promotion rules — not move *legality*. For that
there's `san.Board`: it tracks piece placement and `Apply`s parsed moves,
rejecting ones with no piece able to make them, a blocked path, or a
capture flag that doesn't match the board (including en passant). It
doesn't yet compute attacked squares, so it won't stop a king from walking
into check, and it trusts the move's own Check/Mate flags rather than
deriving them.

The CLI is parse-and-reprint only for now; it doesn't wire up `Board`.

## Usage

Build it:

    go build -o sanlint .

Feed it movetext on stdin, one game per line:

    echo "e4 e5 Nf3 Nc6 Bb5" | ./sanlint

Strict mode (the default) rejects anything off-canon:

    $ echo "0-0" | ./sanlint
    san: invalid move "0-0": does not match SAN move syntax

    $ echo "0-0" | ./sanlint --lenient
    O-O

Lenient mode currently normalizes:

- digit castling (`0-0`, `0-0-0`) to letter-O castling (`O-O`, `O-O-O`)
- a missing `=` before a promotion piece (`e8Q` -> `e8=Q`)
- trailing annotation glyphs (`Nf3!?`, `Qxb7??`)
- a trailing `e.p.` en passant marker

Whatever comes out the other side of parsing, in either mode, prints back in
canonical strict form. That's the pretty-printer half: pipe a messy PGN
through sanlint and get consistently formatted SAN back. Move numbers
(`1.`, `12...`) and game termination markers (`1-0`, `1/2-1/2`, `*`) in the
input line are skipped rather than treated as moves.

## As a library

    import "github.com/mjjones7/sanlint/san"

    mv, err := san.Parse("exd8=Q+", false)
    if err != nil {
        // strict mode rejected it
    }
    fmt.Println(mv.String()) // "exd8=Q+"

Checking a move against an actual position:

    board := san.NewBoard()
    mv, _ := san.Parse("Nf3", false)
    if err := board.Apply(mv); err != nil {
        // no white knight can reach f3, or some other board-level problem
    }

## License

MIT, see LICENSE.
