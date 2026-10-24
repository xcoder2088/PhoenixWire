# Security policy

Phoenix Wire moves real money (NIM, USDC, USDT). Treat it as such.

## Reporting a vulnerability

**Do not open a public issue for a security problem.** Use GitHub's private vulnerability reporting for this repository
(*Security → Report a vulnerability*). Describe what you found, how to reproduce it, and what it could do. If private reporting is not
available, open an issue that says only "security contact requested" and no detail: a maintainer will reply with a private channel.

This is a small, owner-run project: there is no bug bounty and no guaranteed response time, but reports are read and fixed first.

## What is in scope

The Go gateway and engine (`internal/`, `engine/secureclaim/`), the sender and recipient pages (`internal/web/static/`), and the
deployment files (`deploy/`, `scripts/`). Out of scope: the owner's own server and its configuration, third-party services (mail provider,
RPC nodes, price feeds), and the vendored third-party code (report that upstream).

## Known limits you should read before relying on it

- An independent review is in [docs/AUDIT-2026-10-24.md](docs/AUDIT-2026-10-24.md): the findings, what was done, and what is left open.
  It is a code review, **not a certification**: no formal proof, no cryptographic audit of the libraries or of the Safe bytecode.
- Chain facts (balances, funding, prices, gas) come from public RPC nodes and a price feed: they are trusted third-party data, not independently
  verified consensus. Run your own node for anything that matters.
- A card's USDC/USDT money sits in a two-key Safe: the sender's browser keeps one key. Losing that browser's data means losing the ability to
  take the money back by hand; the server never holds that key.
- Never commit secrets. `.env`, `*.key`, `*.pem`, state and vault directories are ignored on purpose; the master key and the state belong together
  and need an encrypted backup ([docs/DEPLOIEMENT-SIMPLE.md](docs/DEPLOIEMENT-SIMPLE.md)).
