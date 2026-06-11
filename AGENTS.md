# Planning & Specs
- NEVER commit plans or specs to the repository
- NEVER commit while implementing plans in the superpowers skill or any other skill

# Building
- ALWAYS build after making code changes
- To build, run `go build -o blvckhole .` — this writes the `./blvckhole` artifact that `~/.local/bin/blvckhole` symlinks to
- Do NOT rely on `go build ./...` to test changes — it only compile-checks and discards the binary, so the installed/symlinked binary stays stale
