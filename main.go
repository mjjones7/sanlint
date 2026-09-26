// Command sanlint reads chess movetext from stdin, one game per line, and
// prints each move back out in canonical SAN, or reports why it couldn't be
// parsed or why it isn't a legal move in the position it was played in.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mjjones7/sanlint/san"
)

var moveNumberRe = regexp.MustCompile(`^\d+\.+$`)

var terminationMarkers = map[string]bool{
	"1-0": true, "0-1": true, "1/2-1/2": true, "*": true,
}

func main() {
	lenient := flag.Bool("lenient", false, "accept common non-canonical SAN spellings "+
		"(0-0 for O-O, a missing '=' before promotion, trailing !/? annotations, "+
		"e.p. suffixes) before validating")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	exitCode := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		board := san.NewBoard()
		for _, tok := range strings.Fields(line) {
			if moveNumberRe.MatchString(tok) || terminationMarkers[tok] {
				continue
			}
			mv, err := san.Parse(tok, *lenient)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				exitCode = 1
				continue
			}
			if err := board.Apply(mv); err != nil {
				fmt.Fprintf(os.Stderr, "san: illegal move %q: %v\n", tok, err)
				exitCode = 1
				continue
			}
			fmt.Println(mv.String())
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "sanlint: reading input:", err)
		os.Exit(2)
	}
	os.Exit(exitCode)
}
