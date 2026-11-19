# Documentation plan (temporary)

Status: temporary. Delete this file when the mdbook is complete.  
Scope: public documentation for the whole project (server, client, content,
tools and Lisp).  
Book language: English. Changelog language: English.

## 1. Goal

A new operator, content author or developer should be able to:

- install and run the server from a clean checkout;
- create accounts/characters and log in with the patched client;
- understand the architecture and the protocol;
- use every admin tool safely;
- create quests, items, maps, events and Lisp formulas/scripts;
- run tests, load tests and a production distribution;
- troubleshoot the common failure modes.

The book is the reference. `README.md` becomes a short entry point that links to
the book instead of duplicating it.

## 2. Deliverables

- `CHANGELOG.md` at the repository root, Keep a Changelog + SemVer format.
- `book/` with mdBook sources:
  - `book/book.toml`
  - `book/src/SUMMARY.md`
  - chapter Markdown files
- Optional: `book/src/images/` only for diagrams that cannot be expressed as
  text. No screenshots.
- Updated root/server READMEs that point to the book.
- This plan removed after the book builds and is reviewed.

## 3. Audience and tone

- Operators: commands, config, backups, security, troubleshooting.
- Content authors: RPG Maker data, TOML/CSV sources, Lisp, events, quests.
- Developers: package layout, protocol, actor model, tests, extension points.
- English, second person, short paragraphs, copy-pasteable commands.
- Windows-first commands (PowerShell) because the client is RPG Maker VX Ace on
  Windows. Mention Linux where the Go server runs unchanged.

## 4. Proposed book structure

### Part I — Getting started

1. **Introduction**
   - what VXA-OS is, what is replaced, current status and limitations.
2. **Requirements**
   - Windows client, Go toolchain, SQLite, optional Zig for `VXAOS.dll`,
     repository layout.
3. **Initial setup**
   - copy config, migrate, create admin account/character, apply client
     patches, build content, start server, first login.
4. **Distribution**
   - `just dist`, `dist/vxaos-server-*.7z`, what the archive contains, start
     script, updating a live server.

### Part II — Running a server

5. **Configuration reference**
   - `config.toml` server/auth/store fields;
   - shared `content/configs.toml` and `content/enums.toml`;
   - ports, bind address, tick, save interval, map idle, AOI radius/hysteresis,
     connection limits.
6. **Accounts and characters**
   - `vxaos-admin` commands, slots, groups, VIP, bans, deletion.
7. **Admin tools**
   - in-game admin panel commands and permission levels;
   - kick, teleport, go, pull, item/weapon/armor/gold, ban IP/account, unban,
     global switch, MOTD, mute, global message;
   - safety notes and audit expectations.
8. **Operations**
   - logs, graceful shutdown, persistence worker, backups, migrations,
     SQLite/MySQL/Postgres notes, reverse proxy/VPN because there is no TLS.
9. **Performance and scaling**
   - map actors, AOI, on-demand maps, route cache, load tool;
   - how to read the load reports and tune `aoi_radius_tiles`,
     `map_idle_seconds`, tick and save interval.

### Part III — Content authoring

10. **Content overview**
    - what lives in `.rvdata2`, what lives in TOML/CSV, and what is generated.
11. **Items, weapons, armors, classes, skills, enemies and states**
    - RPG Maker data fields that the server actually reads;
    - note tags, formulas, drop tables, state features;
    - validation checklist.
12. **Quests**
    - `content/quests.toml` schema, requirements, kills/items, rewards,
      repeatable quests, client text;
    - `vxaos-content check/build`.
13. **Maps, events and common events**
    - event pages, triggers, conditions, move routes, touch/autorun;
    - transfers, shops, banks, teleports, guild creation;
    - server-side vs client-only commands and `[NG]`.
14. **Vocab, Configs and Enums**
    - `content/locales/*.csv`, `content/configs.toml`,
      `content/enums.toml`;
    - generation into `Scripts.rvdata2`, validation and parity tests.
15. **Items, drops and the economy**
    - inventory stack rules, soulbound items, bank limits, ground drops,
      pickup ownership/party rules, trade restrictions.

### Part IV — Lisp

16. **The Lisp VM**
    - why the server uses Lisp, where it runs, what it never exposes to the
      client;
    - reader, compiler, stack VM, closures, variadics, native registry, host
      objects, instruction/depth limits.
17. **Damage formulas**
    - `a`/`b`/`v` objects, supported parameters, examples, error reporting,
      `vxaos-formula check/apply`.
18. **Event scripts**
    - command 355 migration, supported natives, shop/bank/quest/guild hooks,
      `[NG]`, `vxaos-events check/apply`;
    - examples for quests, checkpoints, chat, VIP and graphics.
19. **Lisp reference**
    - syntax, special forms, built-ins, native function signatures, limits,
      debugging and common mistakes. Link to `server/docs/lisp.md` as the
      low-level ABI reference.

### Part V — Development

20. **Architecture**
    - client/server split, protocol v2, coordinator, map actors, AOI,
      persistence worker, instance keys;
    - concurrency invariants and why map state is closed.
21. **Protocol v2**
    - handshake, framing, fragmentation, auth challenge/proof, opcodes,
      errors, versioning and how to add a message.
22. **Repository and workflow**
    - package map, build/test/vet/race, client patch application, content
      generation, load test, dist.
23. **Extending the server**
    - adding a client opcode, a world command, a map-local command, a
      persistence field, a native Lisp function;
    - tests and race expectations.

### Part VI — Reference

24. **Troubleshooting and FAQ**
    - client cannot connect, version mismatch, auth failure, character already
      online, DB locked, client patch not applied, empty map/AOI, send queue,
      missing content.
25. **Command reference**
    - every `vxaos-admin`, `vxaos-content`, `vxaos-clientpatch`,
      `vxaos-formula`, `vxaos-events`, `vxaos-load` and `vxaos-input` command.
26. **Glossary and changelog link**

## 5. Source of truth

The book must be verified against the code, not copied from old README claims:

- `server/internal/config/config.go`, `config.example.toml`,
  `content/configs.toml`, `content/enums.toml`
- `server/cmd/vxaos-admin`, `vxaos-content`, `vxaos-clientpatch`,
  `vxaos-formula`, `vxaos-events`, `vxaos-load`, `vxaos-input`
- `server/internal/game` for gameplay rules and admin commands
- `server/docs/lisp.md` and `docs/protocol-v2.md`
- `server/docs/load-test-*.md` for measured numbers
- `server/client-patches/manifest.toml` for the client patch list

Every command in the book must be run once in a clean environment before the
chapter is considered done. Config examples must match the real loader.

## 6. Work order

1. Changelog (`CHANGELOG.md`).
2. mdBook scaffold and theme/SUMMARY.
3. Part I (setup, distribution).
4. Part II (config, admin, operations, performance).
5. Part III (content, quests, maps, vocab, economy).
6. Part IV (Lisp) — the largest and most important chapter.
7. Part V (architecture, protocol, development).
8. Part VI (troubleshooting, command reference, glossary).
9. Root/server README cleanup and links.
10. `mdbook build`, command verification, review, remove this plan.

## 7. Acceptance criteria

- `mdbook build book` succeeds with no warnings.
- Every command block was executed or explicitly marked as illustrative.
- Every config key and admin command in the book exists in the code.
- The Lisp chapter covers formulas, event scripts, natives, limits and examples.
- A new operator can go from clone to logged-in character using only the book.
- Changelog matches the released tag and links to the book.
- No screenshots; diagrams are text or generated images only where necessary.
- Temporary plan deleted.

## 8. Open choices (decide before scaffolding)

- Book location: `book/` at repository root (recommended).
- Version for the first public release: `3.0.0` (major 3 starts over the v2
  server; protocol v2 remains the wire protocol).
- Hosting: GitHub Pages via mdBook later; not required for the first build.
- Whether the old Ruby server remains documented as historical reference or is
  out of scope (recommended: one short historical note, no operational guide).
