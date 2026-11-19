# VXA-OS

VXA-OS is an online multiplayer platform for RPG Maker VX Ace. It pairs a
patched VX Ace client with a dedicated Go server for accounts, characters,
persistent worlds, quests, events and social play.

## Repository layout

| Path | Contents |
|---|---|
| `client/` | Patched VX Ace game, data, graphics and native library |
| `server/` | Go server, admin tools and content pipeline |
| `server-rb/` | Legacy Ruby server and historical assets (reference only) |
| `book/` | English documentation built with mdBook |
| `extra/` | Additional tools, DLL sources and historical material |
| `docs/` | Temporary documentation plan (removed when the book is complete) |
| `CHANGELOG.md` | Release notes |

## Documentation

The full manual is in [`book/`](book/). It covers installation, server
configuration, admin tools, content authoring, quests, the Lisp scripting
language and development.

```powershell
cargo install mdbook --locked   # once
mdbook serve book               # local preview
mdbook build book               # static site in book/build
```

If you are new here, read the book in order:

1. Introduction
2. Requirements
3. Initial setup

## Releases

The current release is **3.0.0**. See [`CHANGELOG.md`](CHANGELOG.md) for the
complete list of features, fixes and compatibility notes.

## Status

The Go server is the active implementation. The legacy Ruby server is kept for
historical reference and is not required to run VXA-OS.
