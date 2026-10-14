// calc is a REPL for arithmetic. It reads Go expressions, computes them
// with exact arithmetic, and prints the result.
//
//	calc            # start the REPL
//	calc '7 / 2'    # print one result and exit
//	echo 2+2 | calc # read one expression per line from stdin
package main

import (
	"bufio"
	"errors"
	"fmt"
	"go/constant"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
)

const (
	prompt      = "> "
	historyFile = ".calc_history"
)

func main() {
	if len(os.Args) > 1 {
		v, err := eval(strings.Join(os.Args[1:], " "), nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(format(v))
		return
	}

	// A pipe or a file on stdin gets no prompt and no line editing.
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		if err := run(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := repl(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// repl reads lines from the terminal. liner gives the line editing:
// up and down arrows walk the history, Ctrl+A and Ctrl+E move to the
// start and end of the line, Ctrl+R searches the history, Ctrl+C
// discards the line, and Ctrl+D on an empty line exits.
//
// The history lives in ~/.calc_history, as irb keeps ~/.irb_history,
// so the up arrow reaches lines from an earlier session.
//
// liner and not golang.org/x/term: term.Terminal returns io.EOF for
// Ctrl+C before it consumes the key, so the next ReadLine reads the
// same bytes again.
func repl() error {
	ln := liner.NewLiner()
	defer ln.Close()
	ln.SetCtrlCAborts(true)

	path := ""
	if home, err := os.UserHomeDir(); err == nil {
		path = filepath.Join(home, historyFile)
		if f, err := os.Open(path); err == nil {
			ln.ReadHistory(f)
			f.Close()
		}
	}

	var prev constant.Value
	for {
		line, err := ln.Prompt(prompt)
		if errors.Is(err, liner.ErrPromptAborted) {
			continue
		}
		if errors.Is(err, io.EOF) {
			fmt.Println()
			break
		}
		if err != nil {
			return fmt.Errorf("reading line: %w", err)
		}
		if strings.TrimSpace(line) != "" {
			ln.AppendHistory(line)
		}
		out, v, quit := step(line, prev)
		if quit {
			break
		}
		if v != nil {
			prev = v
		}
		if out != "" {
			fmt.Println(out)
		}
	}

	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("saving history: %w", err)
	}
	defer f.Close()
	if _, err := ln.WriteHistory(f); err != nil {
		return fmt.Errorf("saving history: %w", err)
	}
	return nil
}

// run reads one expression per line from r and writes each result to
// w. It is the REPL without a terminal, for a pipe or a test.
func run(r io.Reader, w io.Writer) error {
	var prev constant.Value
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		out, v, quit := step(sc.Text(), prev)
		if quit {
			return nil
		}
		if v != nil {
			prev = v
		}
		if out != "" {
			fmt.Fprintln(w, out)
		}
	}
	return sc.Err()
}

// step computes one line. It returns the text to print, the value for
// _ to read next (nil when the line gave none), and whether the line
// asks to exit.
func step(line string, prev constant.Value) (string, constant.Value, bool) {
	line = strings.TrimSpace(line)
	switch line {
	case "":
		return "", nil, false
	case "exit", "quit":
		return "", nil, true
	}
	v, err := eval(line, prev)
	if err != nil {
		return "error: " + err.Error(), nil, false
	}
	return format(v), v, false
}
