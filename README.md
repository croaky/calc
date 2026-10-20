# calc

`calc` is a REPL for arithmetic. It reads Go expressions and computes
them with exact arithmetic.

Install:

```bash
go install github.com/croaky/calc@latest
```

Start the REPL:

```txt
$ calc
> 2 + 2
4
> (12 + 4) * 3 / 2.5
19.2
> 7 / 2
3.5
> 0.1 + 0.2
0.3
> _ * 10
3
> 1 << 70
1180591620717411303424
> quit
```

Give an expression as arguments to print one result and exit:

```bash
calc '7 / 2'
```

Pipe expressions on stdin, one per line, to print one result per line:

```bash
printf '2 + 2\n_ * 10\n' | calc
```

## Expressions

An expression is a Go constant expression: integer and floating-point
literals (`1_000`, `0xff`, `1e-3`), parentheses, and the operators
`+ - * / % & | ^ &^ << >>`. `%`, the bitwise operators, and the shifts
need integers.

`/` gives the exact quotient. `7 / 2` is `3.5`, not Go's truncated
`3`.

`calc` holds each value as an exact integer or rational, as the Go
compiler holds a constant. So `0.1 + 0.2` is `0.3`, and an integer
keeps every digit. `calc` rounds to a `float64` only to print a
fraction.

`_` is the result of the last line that gave one, as in irb.

## Keys

- Up and Down: walk the history.
- Ctrl+A and Ctrl+E: move to the start and the end of the line.
- Ctrl+R: search the history.
- Ctrl+C: discard the line.
- Ctrl+D on an empty line, `exit`, or `quit`: exit.

`calc` saves the history in `~/.calc_history`, so the up arrow reaches
lines from an earlier session.

`calc` depends on [peterh/liner](https://github.com/peterh/liner) for
line editing.

## Developing

`Checkfile` lists the checks CI runs. `AGENTS.md` gives the commands to
run before a commit, and the commit message rules.

## GitHub repo is a mirror

Development happens on [sockeye](https://sockeye.sh), a self-hosted
review and CI server, which holds in-progress branches. GitHub receives
`main` so `go install` works.

## License

MIT
