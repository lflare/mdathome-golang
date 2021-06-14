# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added

### Changed

### Deprecated

### Removed

### Fixed
- [2021-06-14] Fixed reverse proxy `X-Forwarded-For` IP handling by [@lflare].

### Security

## [v1.11.4] - 2021-06-10
### Fixed
- [2021-06-10] Properly refixed TLS SNI random crashes again by [@lflare].

## [v1.11.3] - 2021-06-07
### Changed
- [2021-06-02] Removed `upx` from macOS/Darwin builds by [@lflare].

### Fixed
- [2021-06-07] Fixed TLS SNI random crashes by [@lflare].

## [v1.11.2] - 2021-05-30
### Added
- [2021-05-30] Very rudimentary SNI whitelist support by [@lflare].

### Fixed
- [2021-05-30] Fixed headers change in specification 31 by [@lflare].
- [2021-05-30] Use clientSettings for HTTP/2 setting by [@lflare].

## [v1.11.1] - 2021-05-29
### Fixed
- [2021-05-29] Fixed incorrect size report override by [@lflare].

## [v1.11.0] - 2021-05-29
### Added
- [2021-05-28] Added `send_server_header` parameter to disable sending the `Server` header by [@Korbeil].
- [2021-05-29] Added some form of reverse proxy IP middleware by [@lflare].
- [2021-05-29] Added rudimentary settings version migrator by [@lflare].

### Changed
- [2021-05-29] Reworked configuration settings to be on a per-category basis by [@lflare].
- [2021-05-29] Updated client specification to 31 by [@lflare].

## [v1.10.3] - 2021-05-13
### Added
- [2021-05-13] Added configuration option in JSON for specifying logs folder by [@lflare].

## [v1.10.2] - 2021-05-13
### Added
- [2021-05-13] Commandline flag to read configuration from specific path by [@lflare].

### Fixed
- [2021-05-13] Bind specifically to IPv4 ports by [@lflare].

## [v1.10.1] - 2021-04-16
### Changed
- [2021-04-16] Made API backend configurable by [@lflare].

### Fixed
- [2021-04-16] Redid project linting and formatting guidelines with staticcheck by [@lflare].

## [v1.10.0] - 2021-03-18
### Changed
- [2021-03-11] Added more logging fields in JSON structure by [@lflare].
- [2021-03-12] Updated to client specification 30 by [@lflare].

### Removed
- [2021-03-12] Removed test chapter exemptions by [@lflare].

## [v1.9.3] - 2021-03-05
### Changed
- [2021-02-28] Updated `go.mod` with Golang 1.16 by [@lflare].

### Fixed
- [2021-02-25] Fixed missing `f` thanks to LittleEndu by [@lflare].
- [2021-03-05] Fixed IPv6 issue with backend communication by [@lflare].

## [v1.9.2] - 2021-02-25
### Changed
- [2021-02-13] Updated README.md with more up-to-date instructions by [@lflare].
- [2021-02-25] Recompiled with Golang v1.16 by [@lflare].

## [v1.9.1] - 2021-02-02
### Added
- [2021-01-28] Added low-memory mode option to stream images straight from disk by [@lflare].
- [2021-02-02] Added back ALL THE COMPILATIONS by [@lflare].

### Changed
- [2021-01-28] Reworked diskcache to stream files more efficiently by [@lflare].
- [2021-02-02] Added `disable_tokens` handling from backend by [@lflare].
- [2021-02-02] Updated to client specification 23 by [@lflare].
- [2021-02-02] Lowered startup delay to 5 seconds on older versions of client by [@lflare].

## [v1.9.0] - 2021-01-24
### Added
- [2021-01-21] Added Prometheus metrics of diskcache by [@lflare].
- [2021-01-23] Added 15 seconds upstream timeout by [@lflare].
- [2021-01-23] Added experimental geoip support to Prometheus metrics by [@lflare].

### Changed
- [2021-01-21] Adjusted logging of diskcache by [@lflare].
- [2021-01-23] Made server read/write timeouts more aggresive with 30s and 1m respectively by [@lflare].
- [2021-01-23] Properly pre-processed IP address to only log IP addresses without ports by [@lflare].
- [2021-01-24] Reworked for auto-downloading of MaxMind databases for geolocation by [@lflare].

## [v1.8.1] - 2021-01-10
### Added
- [2021-01-20] Allow overriding of reported address to backend by [@lflare].

### Changed
- [2021-01-10] Increased interval of refresh & backend ping to 30 seconds by [@lflare].
- [2021-01-10] Comply with specification version 20 and default to verify tokens by [@lflare].
- [2021-01-20] Decreased interval of refresh and server ping back to 10 seconds by [@lflare].

### Removed
- [2021-01-15] Removed intermediary and stream image direct from cache to visitor by [@lflare].

## [v1.8.0] - 2021-01-10
### Added
- [2021-01-04] Added option for overriding port advertisement made to backend server by [@lflare].
- [2021-01-10] Added token whitelist for client specification compliance by [@lflare].
- [2021-01-10] Updated to client specification version 20 by [@lflare].

### Changed
- [2021-01-07] Heavily refactored Prometheus metric labels for clarity by [@lflare].

## [v1.7.6] - 2021-01-04
### Fixed
- [2021-01-04] Fixed streamed images Content-Length header being inaccurate on `data-saver` images by [@lflare].

## [v1.7.5] - 2021-01-04
### Added
- [2021-01-04] Adding Prometheus metric for invalid checksum images by [@lflare].

## [v1.7.4] - 2021-01-04
### Fixed
- [2021-01-04] Disabled image verification for `data-saver` images by [@lflare].

## [v1.7.3] - 2021-01-03
### Fixed
- [2021-01-03] Fixed Last-Modified header reporting by [@lflare].

## [v1.7.2] - 2021-01-03
### Changed
- [2021-01-03] Improved goreleaser configuration by [@lflare].

### Fixed
- [2021-01-03] Get diskcache to work with logrus logger by [@lflare].

## [v1.7.1] - 2021-01-03
### Changed
- [2021-01-03] Swapped to VictoriaMetrics for better Histogram by [@lflare].

## [v1.7.0] - 2021-01-03
### Added
- [2021-01-03] Added Prometheus metrics endpoint by [@lflare].

### Changed
- [2021-01-03] Organised settings by type by [@lflare].

## [v1.6.2] - 2021-01-01
### Changed
- [2021-01-01] Changed timestamp format to RFC3339 instead of RFC822 by [@lflare].

### Fixed
- [2021-01-01] Fixed file logging not updating log level by [@lflare].

## [v1.6.1] - 2020-12-29
### Fixed
- [2020-12-29] Fixed invalid logging on invalid token but not rejected requests by [@lflare].

## [v1.6.0] - 2020-12-29
### Added
- [2020-12-29] Added option to disable upstream connection pooling by [@lflare].

### Changed
- [2020-12-29] Revamped logging system with loglevels and more by [@lflare].

## [v1.5.5] - 2020-12-22
### Changed
- [2020-12-22] Added configuration option for upstream override by [@lflare].

## [v1.5.4] - 2020-10-12
### Changed
- [2020-10-12] Replaced boltdb implementation with etcd's by [@lflare].

## [v1.5.3] - 2020-09-29
### Added
- [2020-09-29] Added client configuration of allowing visitor-forced image refresh by [@lflare].

## [v1.5.2] - 2020-09-08
### Added
- [2020-09-08] Added client configuration of optional HTTP2 by [@lflare].
- [2020-08-23] Added some form of image integrity check via use of SHA256 checksums provided by upstream by [@lflare].

### Changed
- [2020-09-08] Bumped client version up to 19 by [@lflare].

## [v1.5.1] - 2020-08-16
### Fixed
- [2020-08-16] Fixed fatalistic logging for cache miss by [@lflare].

## [v1.5.0] - 2020-08-15
### Added
- [2020-08-15] Added argument system by [@lflare].
- [2020-08-15] Added `-shrink-database` argument flag to shrink overly huge cache.db files by [@lflare].

### Changed
- [2020-08-15] Massively refactored code and included diskcache-golang as an internal module by [@lflare].

## [v1.4.1] - 2020-08-15
### Changed
- [2020-08-15] Updated to v0.5.1 of diskcache by [@lflare].

## [v1.4.0] - 2020-08-14
### Added
- [2020-08-14] Added `cache_refresh_age_in_seconds` configuration option to reduce cache update speeds for large caches by [@lflare].

### Changed
- [2020-08-14] Updated to v0.5.0 of diskcache by [@lflare].
- [2020-08-14] Massively refactored codebase by [@lflare].

## [v1.3.2] - 2020-08-09
### Changed
- [2020-08-07] Swapped out retryablehttp for default vanilla http.Client for keep-alive reuse by [@lflare].

## [v1.3.1] - 2020-08-01
### Fixed
- [2020-08-01] Updated filename regex for more flexibility in image filenames by [@lflare].

## [v1.3.0] - 2020-07-19
### Added
- [2020-07-18] Added version checker by [@lflare].

### Fixed
- [2020-07-18] Fixed incorrect reported disk space to server for edge cases by [@lflare].

## [v1.2.4] - 2020-07-18
### Added
- [2020-07-18] Added `make local` support for development builds by [@lflare].

### Fixed
- [2020-07-18] Dropped connections no longer save half-corrupted images to cache by [@lflare].

### Changed
- [2020-07-18] Properly refactored code to fit golangci-lint styles with advisory from @columna1 by [@lflare].

## [v1.2.3] - 2020-07-14
### Added
- [2020-07-14] Added image verification code by [@lflare].

### Fixed
- [2020-07-10] Fixed invalid response code for invalid tokens due to typo by [@lflare].

## [v1.2.2] - 2020-07-09
### Changed
- [2020-07-09] Add client spec version to Server header sent by client by [@lflare].

## [v1.2.1] - 2020-07-09
### Changed
- [2020-07-09] Increased WriteTimeout to 5 minutes to match token expiration timing by [@lflare].
- [2020-07-09] Bumped version number to 16 to match 1.1.5 official build by [@lflare].

## [v1.2.0] - 2020-07-05
### Added
- [2020-07-05] Added rudimentary validation of request tokens by [@lflare].
- [2020-07-05] Automatic update of client settings in the event of new fields by [@lflare].
- [2020-07-05] Added version numbers to build artifacts by [@lflare].

### Changed
- [2020-07-03] Updated README.md with relevant up-to-date information by [@lflare].
- [2020-07-03] Updated client defaults by [@lflare].
- [2020-07-04] Changed graceful shutdown timer from 15 to 30 seconds by [@lflare].
- [2020-07-04] Updated Makefile for single builds to produce static binaries by [@lflare].
- [2020-07-05] Convert `sanitized_url` to `sanitizedUrl` for better cohesion by [@lflare].

### Fixed
- [2020-07-04] Reduced aborted requests due to faulty timer updating by [@lflare].

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

[Unreleased]: https://github.com/lflare/mdathome-golang/compare/v1.5.1...HEAD
[v1.5.1]: https://github.com/lflare/mdathome-golang/compare/v1.5.0...v1.5.1
[v1.5.0]: https://github.com/lflare/mdathome-golang/compare/v1.4.1...v1.5.0
[v1.4.1]: https://github.com/lflare/mdathome-golang/compare/v1.4.0...v1.4.1
[v1.4.0]: https://github.com/lflare/mdathome-golang/compare/v1.3.2...v1.4.0
[v1.3.2]: https://github.com/lflare/mdathome-golang/compare/v1.3.1...v1.3.2
[v1.3.1]: https://github.com/lflare/mdathome-golang/compare/v1.3.0...v1.3.1
[v1.3.0]: https://github.com/lflare/mdathome-golang/compare/v1.2.4...v1.3.0
[v1.2.4]: https://github.com/lflare/mdathome-golang/compare/v1.2.3...v1.2.4
[v1.2.3]: https://github.com/lflare/mdathome-golang/compare/v1.2.2...v1.2.3
[v1.2.2]: https://github.com/lflare/mdathome-golang/compare/v1.2.1...v1.2.2
[v1.2.1]: https://github.com/lflare/mdathome-golang/compare/v1.2.0...v1.2.1
[v1.2.0]: https://github.com/lflare/mdathome-golang/compare/v1.1.0...v1.2.0
[v1.1.0]: https://github.com/lflare/mdathome-golang/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/lflare/mdathome-golang/releases/tag/v1.0.0
