# Planning & Specs
- NEVER commit plans or specs to the repository
- NEVER commit while implementing plans in the superpowers skill or any other skill

# Sandbox
- You might be running inside a sandbox where some commands (e.g. `zellij`, host-only tools) are unavailable or behave differently
- Check with the `IS_SANDBOX` env var: if `IS_SANDBOX=1` you are sandboxed
- When a command fails because you are sandboxed, ask the user to run it on their host and paste the output back — do not assume the command is broken

# Building
- ALWAYS build after making code changes
- To build, run `go build -o blvckhole .` — this writes the `./blvckhole` artifact that `~/.local/bin/blvckhole` symlinks to
- Do NOT rely on `go build ./...` to test changes — it only compile-checks and discards the binary, so the installed/symlinked binary stays stale
- NEVER write the real `./blvckhole` artifact from inside the sandbox (`IS_SANDBOX=1`): the sandbox is Linux, so the binary cannot run on the user's Mac, and it overwrites the host's working artifact that `~/.local/bin/blvckhole` symlinks to. The user builds and runs on their Mac.
- To compile-check inside the sandbox, build to a throwaway path instead, e.g. `go build -o /tmp/blvckhole .` (or `go vet ./...`) — never `-o blvckhole`/`-o ./blvckhole`
