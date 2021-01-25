# MangaDex@Home... in Golang! - [![Total Downloads](https://img.shields.io/github/downloads/lflare/mdathome-golang/total)](https://github.com/lflare/mdathome-golang/releases)
_Another unofficial client-rewrite by @lflare!_

## Disclaimer
No support will be given with this unofficial rewrite. Feature recommendations are not welcomed, but pull requests with new features are. This fork was created entirely out of goodwill and boredom, and if the creator so decides, will not receive future support at any point in time.

## Installation
In order to get this client working, you will need the basic client requirements stipulated by the official MDClient. With that said, head on over to [the releases page](https://github.com/lflare/mdathome-golang/releases) to download a pre-compiled version for your operating system & architecture!

#### Running
To run the client, ensure permissions are set correctly on your operating system, and run it like you would any other binary

```bash
$ ./mdathome_linux_amd64 
2020-07-03T12:34:44+08:00 Failed to read client configuration file, creating anew: open settings.json: no such file or directory
2020-07-03T12:34:44+08:00 Created sample settings.json! Please edit it before running again!
```

Do not worry, this is just the client creating a sample configuration, as it is not able to find one! Open up `settings.json` and edit your configuration as you see fit. Once your client secret is filled in, relaunch the client again in `tmux` or via `systemd` and let the manga flow!

#### Manual Compilation
If you fancy yourself a hardcore masochist that likes to compile everything yourself manually, feel free to do so!

```bash
go get github.com/lflare/mdathome-golang
go run github.com/lflare/mdathome-golang
```

#### Docker
If you want to run the client in a Docker container you can do so! Make sure to point the container to your cache and configuration file location.

```bash
docker run -d -v /path/to/your/cache:/mangahome/cache -v /path/to/your/settings.json:/mangahome/settings.json -p 443:443 lflare/mdathome-golang:latest
```

## Configuration
As with the official client, this client reads a configuration JSON file.

```json
{
    "cache_directory": "cache/",
    "client_port": 44300,
    "override_port_report": 0,
    "client_secret": "",
    "graceful_shutdown_in_seconds": 300,
    "max_kilobits_per_second": 10000,
    "max_cache_size_in_mebibytes": 10240,
    "max_reported_size_in_mebibytes": 10240,
    "cache_scan_interval_in_seconds": 300,
    "cache_refresh_age_in_seconds": 3600,
    "max_cache_scan_time_in_seconds": 15,
    "allow_http2": true,
    "allow_upstream_pooling": true,
    "allow_visitor_refresh": false,
    "enable_prometheus_metrics": false,
    "override_upstream": "",
    "reject_invalid_tokens": false,
    "verify_image_integrity": false,
    "log_level": "trace",
    "max_log_size_in_mebibytes": 64,
    "max_log_backups": 3,
    "max_log_age_in_days": 7
}
```

## Configuration Expplanation
***
### Client Configuration
#### - `cache_directory`
Allows configuration of where the cache will be stored at.

#### - `client_port` - Recommended `443`
Allows configuration of whichever port the client will listen on.

#### - `override_port_report`
Allows overriding of reported port to backend. Defaults to `0` for disabled.

#### - `client_secret`
Self-explanatory, this should be obtained from the [MangaDex@Home page](https://mangadex.org/md_at_home).

#### - `graceful_shutdown_in_seconds`
This setting controls how long to wait after SIGINT for readers to switch off your client before giving up while shutting down gracefully.

*** 
### Speed & Cache Configuration
#### - `max_kilobits_per_second`
This setting currently only reports to the backend, and does not actually limit the speed client side.

#### - `max_cache_size_in_mebibytes`
This is the max cache size in mebibytes stored on your disk, do not exceed what is actually possibly storable on your drive.

#### - `max_reported_size_in_mebibytes`
This is the cache size reported to the backend server. This may cause your server to get more shards, but due to the nature of how this will work, setting this variable too high will result in oversubscription and may incur additional disk and/or network traffic. It is **highly** recommended that you set this variable the same as `max_cache_size_in_mebibytes`.

### Cache Scanning Configuration
#### - `cache_scan_interval_in_seconds`
This setting controls the interval in which the cache is scanned and automatically trimmed/evicted when cumulative cache size exceeds `max_cache_size_in_mebibytes`.

#### - `cache_refresh_age_in_seconds`
This setting controls the maximum age allowed for a cache entry before being refreshed. Larger caches may find it more performant to set it to a greater time interval (e.g. 1 day or 1 week).

#### - `max_cache_scan_time_in_seconds`
This setting controls how long the diskcache will take to scan through the database and filesystem for eviction purposes. After the specific set amount of time in seconds, the function just stops iterating and returns.

***
### Additional Feature Configuration
#### - `allow_http2` - Recommended `yes`
Allows the usage of HTTP2 on your MD@H client, unlike the original Java client. May result in better performance.

#### - `allow_upstream_pooling` - Recommended `yes` unless you got a very large client with multiple upstream IP addresses
Allows pooling and re-use of upstream connections. May result in better performance.

#### - `allow_visitor_refresh` - Recommended `no`
This setting controls if visitors should be allowed to force image refreshes through `Cache-Control` header. (e.g. through a CTRL-SHIFT-R on any modern web browser)

#### - `enable_prometheus_metrics`
This setting controls if client metrics should be published on the `/metrics` endpoint of your server. **Note:** All metrics are public and do not require authentication to access. If this does not strike you fancy, either submit a PR to change this behaviour, or disable it entirely.

#### - `override_upstream` - Recommended empty.
This setting allows you to override the upstream server. If you are a normal MD@H user, this setting is not for you and should be left empty.

#### - `reject_invalid_tokens` - Recommended `no`
This setting controls if the cache server should reject all requests with missing or invalid security tokens. At present (2021/01/04), the official client does not enforce token verifications, and thus it is recommended to be off on this client as well.

#### - `verify_image_integrity`
This setting controls if images in cache should be verified with the checksum in the image name. This only applies to `/data/` images due to limitations with upstream.

***
### Log Settings
#### - `log_level`
This setting controls the log level of the client. Unless you run a gargantuan client, it is recommended you keep the log level at INFO or higher like DEBUG/TRACE.

- TRACE: Timing information like TTFBs and request completion times
- DEBUG: Various debugging information like cache misses, hits, etc.
- INFO: Request information like remote visitor address and image URL.
- WARN: Warnings like upstream request errors, or downstream visitor pull errors
- ERROR: ???

#### - `max_log_size_in_mebibytes`
This setting controls the maximum size a log can grow to before it gets rotated to a backup file.

#### - `max_log_backups`
This setting controls how many backup log archives are allowed before the oldest is deleted forever.

#### - `max_log_age_in_days`
This setting controls the maximum age a log can grow to before it is deleted.

## License
[AGPLv3](https://choosealicense.com/licenses/agpl-3.0/)
