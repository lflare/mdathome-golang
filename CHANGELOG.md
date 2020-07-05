# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- [2020-07-05] Added rudimentary validation of request tokens by [@lflare].

### Changed
- [2020-07-03] Updated README.md with relevant up-to-date information by [@lflare].
- [2020-07-03] Updated client defaults by [@lflare].
- [2020-07-04] Changed graceful shutdown timer from 15 to 30 seconds by [@lflare].
- [2020-07-04] Updated Makefile for single builds to produce static binaries by [@lflare].
- [2020-07-05] Convert `sanitized_url` to `sanitizedUrl` for better cohesion by [@lflare].

### Deprecated

### Removed

### Fixed
- [2020-07-04] Reduced aborted requests due to faulty timer updating by [@lflare].

### Security

## [v1.1.0] - 2020-07-03
### Added
- [2020-07-01] Added official CHANGELOG.md file to keep track of changes from v1.0.0 release by [@lflare].
- [2020-07-01] Simple Makefile to batch build for multiple architectures by [@lflare].
- [2020-07-01] Added badge for linking to latest release on GitHub by [@lflare].
- [2020-07-03] Preliminary check for `Cache-Control` header to pull from upstream by [@lflare].
- [2020-07-03] goreleaser for easier publishing of binaries by [@lflare].

### Changed
- [2020-07-01] Updated Makefile for proper Windows executable file extension by [@lflare].
- [2020-07-03] Upgraded lflare/diskcache-golang to v0.2.3 by [@lflare].

## [v1.0.0] - 2020-07-01
### Added
- [2020-07-01] First stable unofficial client public release by [@lflare]

[Unreleased]: https://github.com/lflare/mdathome-golang/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/lflare/mdathome-golang/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/lflare/mdathome-golang/releases/tag/v1.0.0
