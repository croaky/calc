# Security

Tier: single-user local tool.

`calc` runs on a single user's workstation and computes Go arithmetic
expressions from arguments, stdin, or an interactive REPL.

## Adversaries

1. A compromised dependency in `go.mod` that runs code during build,
   test, or execution.
2. Untrusted expressions passed on stdin or arguments that try to
   exhaust CPU or memory (for example, huge shifts or exponents).

## Accepted risks

Resolve scanner findings in these areas as `accepted_risk` instead of
adding defensive code:

- **Local user compromise**: An attacker who already runs commands as
  the user owns the process, environment, and files.
- **History file and environment trust**: `calc` trusts the caller's
  environment variables, home directory, and REPL history file path.
  Symlink or TOCTOU races in the user's own directory are out of scope.
- **Caller-supplied terminal output**: `calc` writes results to stdout
  and errors to stderr for the caller's own terminal.
