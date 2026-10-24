# Contributing

Thank you for helping. By taking part you agree to the
[code of conduct](CODE_OF_CONDUCT.md).

## Questions and bugs

Search the [issues](https://github.com/vardius/go-api-boilerplate/issues)
first. If nothing matches, open one with the template that fits. For a
security problem, don't open an issue: follow [SECURITY.md](SECURITY.md).

For a change bigger than a bug fix, open an issue with the proposal before
writing code, so we can agree on the shape first.

## Making a change

1. Fork the repository and branch from `master`.
2. Read [`AGENTS.md`](../AGENTS.md): the repo map, the rules for the folder
   you're changing, and the commands below. The code bar is in
   [`docs/standards/`](../docs/standards/shared.md). Both apply whether you
   write the code yourself or with an AI agent.
3. Run the stack locally with `docker compose up --wait` (see the
   [quick start](https://vardius.github.io/go-api-boilerplate/start/quick-start/)).
4. Before you push, run everything CI runs:

   ```bash
   make check
   ```

5. Open a pull request against `master` and fill in its template. Keep it to
   one change; unrelated fixes go in their own pull request.

## Documentation

The website lives in [`website/`](../website/). Code samples there are
imported from real files, so a change to a sample is a change to the code.
