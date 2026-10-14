# Agents guide

calc is a REPL for arithmetic. It reads Go expressions and computes
them with exact arithmetic. See `README.md` for the surface.

## Writing

Write every word in ASD-STE100 Simplified Technical English (STE):
Markdown docs, code comments, commit messages, change descriptions,
and replies in an agent conversation.
See <https://en.wikipedia.org/wiki/Simplified_Technical_English>.

STE is a controlled English for technical writing: one meaning per
word, one idea per sentence, and the actor named. It is not a house
style. It exists so every reader reads a sentence the same way. That
includes a tired reader, a reader in a second language, and an agent
that matches on words.

- One idea per sentence. Keep an instruction to 20 words and a
  description to 25.
- Active voice, present tense, and the actor named: say what acts,
  rather than writing "the line is discarded".
- One word, one meaning. Keep a term the same everywhere rather than
  varying it for tone.
- Use the simple verb, not a noun made from it: "compute the
  expression", not "perform computation of the expression".
- Cut what carries nothing: "simply", "just", "note that", "in order
  to".
- Put a warning or a limit before the step it applies to.
- STE limits a sentence, not a text. Keep each sentence short, but
  keep the sentence that defines a term or connects a cause to its
  effect.
- STE permits technical names. Name the operator, the key, or the
  function, not a vague noun such as "the value".

Apply it to prose, not to code: an identifier, a command, and a quoted
error message stay as they are.

## Architecture

One package at the repo root.

- `eval.go` — a line to a value. `go/parser` parses the line as a Go
  expression, and `eval` walks the tree with `go/constant`. `format`
  writes a value for a person to read.
- `main.go` — the three ways in: arguments, a pipe on stdin, and the
  REPL. `step` computes one line for both the pipe and the REPL, so the
  two cannot disagree. `repl` adds line editing and the history file.

`eval` accepts a short list of node kinds and operators, and refuses
the rest by name. Add a feature as a new case with a test, not as a
general path that accepts more than the tests cover.

`/` gives the exact quotient of two integers, not Go's truncated one.
That is the one place calc differs from Go, and `README.md` says so.
Keep it the only one.

## Dependencies

The standard library, plus `github.com/peterh/liner` for line editing
and `github.com/croaky/is` for test assertions. Taking another is a
design decision, not a step.

`golang.org/x/term` was the first choice for line editing. It fails on
Ctrl+C: `Terminal.ReadLine` returns `io.EOF` before it consumes the
key, so the next call reads the same bytes again.

## Checks

The root `Checkfile` is the list, and CI runs it on every push. Run the
same things before committing:

```sh
go run golang.org/x/tools/cmd/goimports@v0.45.0 -local "$(go list -m)" -w .
go vet ./...
go test -trimpath -buildvcs=false -race -cover ./...
go run golang.org/x/tools/cmd/deadcode@v0.45.0 -test ./...
git ls-files -z '*.go' | xargs -0 gopls check -severity=hint
dprint fmt
git ls-files -z '*.sh' 'bin/*' | xargs -0 shellcheck
```

`goimports` and `dprint` write the fix here. Their CI jobs only report,
because a CI job that rewrites source has nowhere to put it.

Install the hook once per clone with `git config core.hooksPath bin`.
The hook runs the Go checks, and only when the push carries a Go file.

## Tests

Red/green TDD. `calc_test.go` holds a table of input and output for
`eval` and a table of input and error message. A new operator or rule
is a new row in each table it touches.

`TestRun` drives `run`, which is the REPL without a terminal. Put a
test of line handling there, not in a test that opens a pty.

To check the keys by hand, run `go run .` in a terminal. A pty from a
script must report a window size. liner reads the column count, and a
pty with zero columns makes liner fall back to a plain read with no
line editing.

Assertions come from `github.com/croaky/is`: `is := is.New(t)`, then
`is.Eq(got, want)`, `is.NoErr(err)`. A table test keeps `t.Errorf`
with the input in the message, so a failure names its row.

## Commits

- Prefix with what the change acts on: `eval:`, `repl:`, `doc:`,
  `ci:`. Not `calc:` — every commit here is calc, so it says nothing.
- Imperative mood, lowercase except proper nouns. Hard-wrap at 72.
- Include _why_, not just _what_. See `git log` for examples.
- Write the subject for a teammate who has not read the code. Name the
  operator, the key, or the output, not a term the change makes up.
- Write the body as a change description (below). The first push of a
  branch with one commit copies the commit message to the change.
- Sign your work with a `Co-Authored-By` trailer.

## Change descriptions

sockeye squashes a change into one commit whose message is the
change's title and description (`soc edit`). Write the description as
that commit message.

Write for an engineer reading it a year from now. They know Go and
Git. They did not see your conversation, and they do not have the diff
open.

Put the most important fact first. A reader who stops after any
paragraph has the most important part so far. Use this order, and
leave out a part that does not apply:

1. The need. Name who uses what, what went wrong or was missing, and
   what the change does about it.
2. What a user sees now. Name the operator, the key, or the output.
   Say what stops happening and what a user does differently.
3. How it works. Name each part at its first mention, with its kind:
   "the `eval` Go function", "the `lint` check". Give the cause of a
   bug or the rule a feature applies.
4. Next steps: a command to run, or a known gap.

Most changes fit in three to five short paragraphs. To keep a
description short and clear:

- Leave out file lists, line counts, and code sizes. The diff shows
  them.
- Leave out options you did not take, each edge case the tests cover,
  and "tests pass". Put a design argument in a code comment.
- Use plain paragraphs. Git strips a `#` line as a comment, so a
  Markdown header disappears.
- Hard-wrap at 72 columns. Don't backslash-escape backticks. Use a
  quoted heredoc such as `<<'EOF'` with `soc edit`.
- Keep `Co-Authored-By` on the commits, not in the description. The
  merge collects the trailers from the commits it squashes.

## Changes

Work happens on a sockeye change. `soc checkout` allocates one and
prints a worktree; `soc edit` sets its title and description. Do the
edit before the code, not after, so the dashboard and `soc show` say
what the change is.

After a push, read the checks with `git push && soc show --wait`
rather than sleeping and then reading. A `soc show` that lands before
the push is recorded reports the previous commit's checks, green,
about the wrong code.

An agent may merge its change when `soc show --wait` reports every
check `ok` on the pushed head. The merge pushes `main` to the GitHub
mirror, which is what `go install` reads.
