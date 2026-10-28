# Building SysInfo Pro

SysInfo Pro is a [Wails v2](https://wails.io) app: a Go backend and a SvelteKit
frontend in a WebView2 window. It builds on Windows 10 or 11 (64-bit) only.

## What you need

| Tool | Version | Why |
|---|---|---|
| Go | the version in `go.mod` (`go 1.26.0`, toolchain `go1.27.3`) | With Go 1.21+ installed, `go` downloads the pinned toolchain itself. |
| Node.js | 22 LTS (20 or newer works) | Builds the frontend (Vite 6, SvelteKit, Tailwind 4). |
| Wails CLI | v2.12.0, built by `scripts/build-wails-cli.ps1` as `wails-go127` | The stock CLI can't read Go 1.26+ code (see below). |
| Windows SDK | any recent Windows 10/11 SDK | Only for the Store package (`makeappx.exe`, `makepri.exe`). |
| WebView2 runtime | built into Windows 11 and current Windows 10 | Runs the app's window. |

## 1. Build the Wails CLI (once per machine)

The stock Wails v2.12.0 CLI bundles an old `golang.org/x/tools` that can't
parse Go 1.26+ source, so `wails build` fails on this project. The script
builds the same CLI version with `golang.org/x/tools v0.52.0` in a throwaway
module and installs it as `wails-go127.exe` in `$(go env GOPATH)\bin`. It never
replaces a stock `wails` install.

```powershell
.\scripts\build-wails-cli.ps1
wails-go127 version   # v2.12.0
```

## 2. Develop

```powershell
cd frontend; npm ci; cd ..
wails-go127 dev
```

- `wails dev` regenerates the Go-to-JavaScript bindings in
  `frontend/src/lib/wailsjs` by itself. Don't run `wails-go127 generate module`
  while dev mode is running, or the two can fight over the files.
- Dev builds use their own single-instance ID (`instance_dev.go`), so dev mode
  runs next to an installed copy of the app.
- To call Go from a browser while dev mode runs, open http://localhost:34115.

## 3. Check before you commit

```powershell
cd frontend
npm run check        # svelte-check (types)
npm run test         # Vitest unit tests
npm run lint         # Prettier
npm run build        # the Go code embeds frontend/build
cd ..
go vet ./...
go test ./...
```

Tests that need the real machine (PowerShell, hotkeys, Docker, performance
counters) are behind the `integration` build tag. Run them on a dev PC with
`go test -tags integration ./...`.

CI (`.github/workflows/ci.yml`) runs the same checks on every push and pull
request, then builds the exe and keeps it as an artifact.

## 4. Build the exe

```powershell
wails-go127 build -o sysinfopro-3.2.0.exe
```

The exe lands in `build/bin`. Use the version from `wails.json`
(`info.productVersion`).

**Never use `-clean`.** It deletes all of `build/bin`, including earlier builds
and packages you may still need.

## 5. Build the Store package (MSIX)

Build the exe first (step 4), with the same version as `wails.json`. Then:

```powershell
.\build\windows\msix\build-msix.ps1 `
    -IdentityName "<Package/Identity/Name from Partner Center>" `
    -Publisher "CN=<publisher ID from Partner Center>" `
    -PublisherDisplayName "Probably Fine Labs"
```

- The three values are in Partner Center under your app > Product management >
  Product identity.
- The script finds the newest Windows SDK with `makeappx.exe`, stages the exe
  and icons, writes `AppxManifest.xml` from the template, builds
  `resources.pri`, and writes `build\bin\SysInfoPro_<version>.0_x64.msix`.
- The package is unsigned on purpose: the Store signs it during
  certification. To install it on your own PC first, sign it with a test
  certificate.
- Store listing text and certification notes are in
  `build/windows/msix/store-listing.md`.

## Release builds in CI

Pushing a tag such as `v3.2.0` runs `.github/workflows/release.yml`. It runs
every check, refuses to continue if the tag doesn't match `wails.json`, builds
the exe and, if the repository variables `MSIX_IDENTITY_NAME`,
`MSIX_PUBLISHER` and `MSIX_PUBLISHER_DISPLAY_NAME` are set, the MSIX. Both are
kept as artifacts. Nothing is published: upload the package in Partner Center.

## Code signing (SignPath)

The GitHub exe is not signed yet. The SignPath Foundation's free open-source
programme declined the first application (October 2026) because the project
did not yet have enough public visibility; reapply once it has more stars,
references and community activity. The Store package doesn't need this:
Microsoft signs it. The release workflow's signing steps below stay switched
off until the variables are set, so they are harmless in the meantime, and
they work the same with a paid SignPath subscription.

One-time setup:

1. Apply at signpath.org with the public repository. The project needs its
   [MIT license](LICENSE), a public release history and the
   [code signing policy](website/code-signing.html) page on the website.
2. Once approved, SignPath gives you an organization ID and sets up a project
   and a signing policy. In the GitHub repository settings add:
   - variables `SIGNPATH_ORGANIZATION_ID`, `SIGNPATH_PROJECT_SLUG`,
     `SIGNPATH_POLICY_SLUG`
   - secret `SIGNPATH_API_TOKEN` (a SignPath CI user token)
3. Push a tag. The release workflow uploads the exe it built, sends it to
   SignPath, waits for your approval and the signature, checks the signature,
   and keeps the signed exe as the artifact.

Until the variables are set, the signing steps are skipped and the release
builds exactly as before. The signed exe shows "SignPath Foundation" as the
publisher; that is how the free programme works.
