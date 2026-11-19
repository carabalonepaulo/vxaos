# Changelog

The authoritative changelog is [`CHANGELOG.md`](https://github.com/carabalonepaulo/vxaos/blob/main/CHANGELOG.md)
at the repository root.

## 3.0.0 — 2026-11-19

Initial public release of the Go server rewrite.

Highlights:

- protocol v2 and PBKDF2 challenge-response auth;
- SQLite default with migrations;
- map actors, route epochs, snapshot handoff and on-demand maps;
- server-authoritative area of interest;
- full gameplay: combat, skills/items, states, events, quests, drops, shops,
  bank, guilds, parties, friends, trade and admin;
- Lisp damage formulas and event scripts;
- generated client content and tooling;
- 500-bot load reports.

Version 3 starts a new major line over the v2 server; breaking changes are
expected during 3.x. The wire protocol remains v2.

See the root changelog for the complete Added/Changed/Fixed/Performance/
Security lists.
