# SysInfo Pro

Plain answers for your Windows PC, built for developers. SysInfo Pro finds out
why your PC is slow, what is filling your drive and what is wrong with your
developer setup, in plain words, with what to try next. Every tool works
without administrator rights, so it runs on a locked-down work laptop too.

[Get it from the Microsoft Store](https://apps.microsoft.com/detail/9pd2g09hcxk3)

## What it does

22 tools in four groups:

- **System:**
  - Specs, Security check and Battery history.
  - Why so slow, which explains what is holding the PC back right now.
  - Hand over, a checklist before you sell or return a PC.
  - What changed, and a short Stress test.
- **Clean up:** Disk & health, Disk usage, Disk growth, Startup, Software
  updates (winget) and Duplicates (to the Recycle Bin, one copy always kept).
- **Developer:**
  - Dev caches and WSL & Docker disk sizes.
  - File locks.
  - Git projects, which finds work that exists only on this PC.
  - Dev doctor (which python, node or java really runs) and a PATH fixer with
    undo.
- **Network:** Speed test, Network doctor (where the connection breaks) and
  Ports.

There's also a widget that stays on top, alerts before a drive fills up, and a
shareable report with personal details left out by default.

No account, no analytics, no ads. See the
[privacy policy](https://sysinfopro.imanandhu.in/privacy.html).

## Building

See [BUILDING.md](BUILDING.md). In short: Go (version from `go.mod`),
Node 22, and the Wails CLI built by `scripts/build-wails-cli.ps1`; then
`wails-go127 build`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for commit conventions and the manual
release checklist. Security problems: see [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) © 2025-2026 Probably Fine Labs.

Releases on GitHub are signed. Free code signing provided by
[SignPath.io](https://signpath.io), certificate by
[SignPath Foundation](https://signpath.org); see the
[code signing policy](https://sysinfopro.imanandhu.in/code-signing.html).
