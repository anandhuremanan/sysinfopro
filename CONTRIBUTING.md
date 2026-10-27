# Contributing

How to build and test: see [BUILDING.md](BUILDING.md).

## Commits

- **One change per commit.** A bug fix, a new tool and a copy change are three
  commits, not one.
- **Say what changed and why** in the subject line, in the imperative, under
  about 72 characters: `Skip git repos whose config can run commands`, not
  `fix` or `3.2.0 release candidate`.
- Add a body when the reason isn't obvious: what was wrong, what the change
  does, anything a reviewer should check.
- Don't commit build output, installers or other binaries (`build/bin`,
  `frontend/build`, `.exe`, `.msix`). `.gitignore` covers the usual ones.
- Run the checks in BUILDING.md before pushing. CI runs them too.

## Releases

1. Set `info.productVersion` in `wails.json` (three parts, like `3.2.0`).
2. Update the "What's new" text in `build/windows/msix/store-listing.md`.
3. Run the manual checklist below on a real PC.
4. Tag the commit `v<version>` (e.g. `v3.2.0`) and push the tag. The release
   workflow builds the exe and the MSIX as artifacts.
5. Upload the MSIX in Partner Center.

## Manual checklist before each release

These tools change or delete things, or depend on other programs whose output
can change. Unit tests cover their rules, but check them by hand on a real PC,
using test folders, never your real data:

- **Dev caches:**
  - search a folder holding a real project with `node_modules` and an
    unrelated program folder (for example VS Code's install folder);
  - only the project's folder is offered;
  - deleting it frees the space shown, and the project restores with
    `npm install`.
- **Duplicates:**
  - make three copies of a file in a test folder;
  - "Select extra copies" leaves one unticked;
  - removing moves the others to the Recycle Bin and they can be restored;
  - the last copy can never be ticked.
- **PATH fix:**
  - add a missing folder to your own PATH;
  - it is flagged and removed, and **Undo** puts it back;
  - entries that use a variable you defined after starting the app are not
    offered.
- **Temporary files:** cleanup shows live progress and can be stopped, files
  changed in the last hour stay, and it refuses to run if TEMP points at a drive
  root or your user folder.
- **Git projects:**
  - search a test folder with a pushed repo, a repo with unpushed commits, and a
    repo whose `.git/config` sets a `filter`;
  - counts are right, and the last one is skipped, never opened.
- **winget updates:**
  - the All apps list loads;
  - one update installs;
  - on a PC where winget is blocked by policy, the message says so instead of
    failing silently.
- Open every screen once in light, dark and AMOLED themes with a bright custom
  accent, and check for clipped text.
