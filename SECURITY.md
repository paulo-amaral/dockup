# Security Policy

## Supported versions

Only the latest release receives security fixes.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting on this repository
(Security tab → "Report a vulnerability"). Do not open public issues
for security problems.

## What dockup does to your system

dockup is transparent about privileged operations:

- Install steps run distro package managers and the official
  `get.docker.com` script; they require root and say so in the UI.
- The hardening step edits `/etc/docker/daemon.json` only after writing
  a timestamped backup next to it, and never overwrites a file it
  cannot parse.
- The audit step is strictly read-only.
- `install.sh` verifies the SHA-256 of every downloaded binary against
  the release's `checksums.txt` before executing anything.
- No telemetry. dockup makes no network calls beyond the package
  repositories and GitHub releases needed to do its job.
