# Security policy

## Reporting a vulnerability

Please report security problems privately, not in a public issue:

- Use GitHub's private vulnerability reporting: open the repository's **Security** tab and choose **Report a vulnerability**.

Include what you found, how to reproduce it, and which version you used (Settings > General > Copy diagnostics gives the version and Windows build). You should get a reply within a week. Once a fix is released, the report can be made public with credit to you, if you want it.

## Supported versions

Only the latest release, from the Microsoft Store or the latest GitHub release, gets security fixes.

## Scope

SysInfo Pro runs as a standard user. The areas that matter most:

- anything that deletes or moves files (Dev caches, Disk & health temporary files, Duplicates)
- ending processes (Processes, Ports, File locks, Why so slow)
- editing the user's PATH
- running git in projects found by Git projects
- network checks (Network doctor, Speed test, version lookups)
- the saved HTML report and its "leave out personal details" option
