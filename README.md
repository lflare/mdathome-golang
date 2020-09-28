# MangaDex@Home... in Golang! - [![Total Downloads](https://img.shields.io/github/downloads/lflare/mdathome-golang/total)](https://github.com/lflare/mdathome-golang/releases)
_Another unofficial client-rewrite by @lflare!_

## Disclaimer
No support will be given with this unofficial rewrite. Feature recommendations are not welcomed, but pull requests with new features are. This fork was created entirely out of goodwill and boredom, and if the creator so decides, will not receive future support at any point in time.

## Installation
In order to get this client working, you will need the basic client requirements stipulated by the official MDClient. With that said, head on over to [the releases page](https://github.com/lflare/mdathome-golang/releases) to download a pre-compiled version for your operating system & architecture!

### Running
To run the client, ensure permissions are set correctly on your operating system, and run it like you would any other binary

```bash
$ ./mdathome_linux_amd64 
2020-07-03T12:34:44+08:00 Failed to read client configuration file, creating anew: open settings.json: no such file or directory
2020-07-03T12:34:44+08:00 Created sample settings.json! Please edit it before running again!
```

Do not worry, this is just the client creating a sample configuration, as it is not able to find one! Open up `settings.json` and edit your configuration as you see fit. Once your client secret is filled in, relaunch the client again in `tmux` or via `systemd` and let the manga flow!

### Manual Compilation
If you fancy yourself a hardcore masochist that likes to compile everything yourself manually, feel free to do so!

```bash
root@af04d92d0b1e:/go# go get github.com/lflare/mdathome-golang
root@af04d92d0b1e:/go# go run github.com/lflare/mdathome-golang
```

## Configuration
As with the official client, this client reads a configuration JSON file.

```json
{
    "cache_directory": "./cache",
    "client_secret": "",
    "client_port": 44300,
    "max_kilobits_per_second": 10000,
    "max_cache_size_in_mebibytes": 10000,
    "max_reported_size_in_mebibytes": 10000,
    "graceful_shutdown_in_seconds": 300,
    "cache_scan_interval_in_seconds": 300,
    "cache_refresh_age_in_seconds": 3600,
    "max_cache_scan_time_in_seconds": 15,
    "reject_invalid_tokens": false
}
```

### `cache_directory`
Self-explanatory

### `client_secret`
Self-explanatory, this should be obtained from the [MangaDex@Home page](https://mangadex.org/md_at_home).

### `client_port` - Recommended `44300`
Self-explanatory, runs the client on the port you specify.

### `allow_http2` - Recommended `yes`
Self-explanatory, allows non-traditional HTTP2 on your MD@H client!

### `max_kilobits_per_second`
This setting currently only reports to the backend, and does not actually limit the speed client side.

### `max_cache_size_in_mebibytes`
This is the max cache size in mebibytes stored on your disk, do not exceed what is actually possibly storable on your drive.

### `max_reported_size_in_mebibytes`
This is the cache size reported to the backend server. This may cause your server to get more shards, but due to the nature of how this will work, setting this variable too high will cause too much file "swapping". It is **highly** recommended that you set this variable the same as `max_cache_size_in_mebibytes`.

### `graceful_shutdown_in_seconds`
This setting controls how long to wait before giving up while shutting down gracefully.

### `cache_scan_interval_in_seconds`
This setting controls the interval in which the cache is scanned and automatically trimmed/evicted when size exceeds `max_cache_size_in_mebibytes`.

### `cache_refresh_age_in_seconds`
This setting controls the maximum age allowed for a cache entry before being refreshed. Larger caches may find it more performant to set it to a greater time interval (e.g. 1 day or 1 week).

### `max_cache_scan_time_in_seconds`
This setting controls how long the diskcache will take to scan through the database and filesystem for eviction purposes. After the specific set amount of time in seconds, the function just stops iterating and returns.

### `reject_invalid_tokens`
This setting controls if the cache server should reject all requests with missing or invalid security tokens.

### `verify_image_integrity`
This setting controls if image integrity should be verified with upstream.

## License
[AGPLv3](https://choosealicense.com/licenses/agpl-3.0/)
