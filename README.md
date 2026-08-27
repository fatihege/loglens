# Loglens

A dependency-free CLI that streams nginx and JSON access logs and displays traffic, latency, and error statistics.

**Status:** work in progress. Note that everything below is the intended behavior at the final version.

## Sample output

```
$ loglens access.log

SUMMARY  access.log
  Read           1,205,699 lines · 341 malformed (0.03%)
  Parsed         1,205,358 entries
  Time range     2024-10-10 00:00:04 → 2024-10-10 23:59:58  (24h 0m)
  Throughput     13.9 req/sec average · 412 peak (14:32)
  Transferred    8.4 GB
  Error rate     2.64%   (27,832 4xx · 4,003 5xx)

STATUS
  200    1,142,318   94.77%   ████████████████████████████
  304       31,205    2.59%   █
  404       21,104    1.75%   █
  400        6,728    0.56%
  500        3,891    0.32%
  502          112    0.01%

TOP PATHS
  #  PATH                   REQS       %      p50      p95      p99   ERR%
  1  /api/products       412,003   34.2%      8ms     42ms    210ms   0.1%
  2  /static/*           298,110   24.7%      1ms      4ms     11ms   0.0%
  3  /api/search         198,442   16.5%     64ms    890ms    3.20s   1.2%
  4  /api/orders          88,301    7.3%    120ms    640ms    2.10s   0.4%
  5  /api/users/{id}      61,224    5.1%     15ms     70ms    198ms   0.0%

LATENCY
  p50      12ms
  p95     184ms
  p99    1.42s
  max    30.00s
```

## Why it exists

A web server writes one line per request. After a day there are millions of lines nobody will ever read. When something breaks (an endpoint times out, doubled error rates after deploy, etc.) there's no practical way to get the answer out of the file. `grep` counts matches but it doesn't compute p99. `awk` can, but it's not intuitive and breaks when the format changes. Hosted platforms exist, but they require to upload your logs which would be a problem on a slow internet and may not be acceptable for privacy reasons. `loglens` is a single binary lives on your machine.

## Install

Requires Go 1.26 or later.

    go install github.com/fatihege/loglens/cmd/loglens@latest

Or build from source:

    git clone https://github.com/fatihege/loglens
    cd loglens
    go build -o loglens ./cmd/loglens

Verify:

    loglens --help

If it isn't found, add `$(go env GOPATH)/bin` to your `PATH`

## Usage

    loglens [flags] [file...]

Reads from stdin when no files are given. Files ending in .gz are decompressed automatically. Multiple files are read in argument order, and the report is order-independent.

### Flags

    --since TIME           only entries at or after TIME
    --until TIME           only entries before TIME
    --status CODE          filter by status (200, 4xx, 5xx)
    --path GLOB            filter by path (/api/*)
    --method METHOD        filter by HTTP method
    --top N                show the N busiest paths (default 5)
    --histogram DUR        request histogram bucketed by DUR (1m, 5m, 1h)
    --json                 emit JSON instead of a table
    --strict               exit non-zero if malformed lines exceed 1%
    --tz OFFSET            timezone for timestamps that carry no offset
    --display-tz OFFSET    timezone for rendering output (default: local)
    --field FIELD=KEY      map a field to a JSON key; repeatable
    --duration-unit UNIT   unit for numeric durations: ns, us, ms, s
    --show-fields          print detected field mapping and exit

Times accept `14:30`, `2024-10-10`, or full RFC3339. Bare times are interpreted in the log's timezone.

Duration precedence: `--duration-unit`, then a unit embedded in the value (`"43ms"`), then inference from the key name, then milliseconds. loglens warns on stderr when it falls back to inference for an ambiguous key.

### Examples

Overview of a day's traffic:

    loglens access.log

Find what was the problem when the site was slow around 2pm:

    loglens --since 14:00 --until 15:00 --status 5xx --top 5 access.log

A log using unusual key names:

    loglens --field path=request_uri --field status=response_code app.jsonl

Nginx-style `request_time` in seconds, exported as a bare number:

    loglens --duration-unit s --field duration=request_time app.jsonl

Server logs in UTC without offsets, read from Istanbul:

    loglens --tz +00:00 access.log

Check what loglens detected before trusting the output:

    loglens --show-fields app.jsonl

Check a single endpoint across a week of rotated logs:

    loglens --path '/api/checkout' /var/log/nginx/access.log.*.gz

Read your app's structured logs from a remote host:

    ssh prod 'cat /var/log/myapp/app.jsonl' | loglens --status 5xx

Find every endpoint with a p99 over one second:

    loglens --json access.log | jq '.paths[] | select(.p99_ms > 1000)'

Fail a CI check if the log format drifted:

    loglens --strict --json access.log > /dev/null

## Supported formats

`loglens` auto-detects the format from the first non-empty line. Lines starting with `{` are parsed as JSON; everything else is treated as nginx text format.

### JSON lines

One JSON object per line, as emitted by `log/slog`, `zerolog`, `zap`, Bunyan, or nginx with `escape=json`.

```json
{"time":"2024-10-10T14:03:11.482+03:00","level":"INFO","msg":"request completed","method":"GET","path":"/api/products","status":200,"bytes":4821,"duration_ms":43,"request_id":"01JAXK2"}
```

Field names vary between loggers, so `loglens` maps common aliases automatically, locking to the first one present in the input:

| Field | Recognised keys |
|---|---|
| Timestamp | `time`, `timestamp`, `ts`, `@timestamp` |
| Method | `method`, `http_method`, `verb` |
| Path | `path`, `url`, `uri`, `request_path` |
| Status | `status`, `status_code`, `http_status`, `code` |
| Bytes | `bytes`, `bytes_sent`, `body_bytes_sent`, `size`, `content_length` |
| Duration | `duration_ms`, `latency_ms`, `duration_us`, `duration_ns`, `duration`, `elapsed`, `latency`, `request_time` |
| Remote address | `remote_addr`, `x_forwarded_for`, `client_ip` |
| Request ID | `request_id`, `trace_id`, `correlation_id` |

Override with `--field FIELD=KEY`, or inspect the resolved mapping with `--show-fields`.

**Timestamps.** Accepted as RFC 3339, `2006-01-02 15:04:05`, or `2006-01-02T15:04:05`; layouts without an offset use `--tz`, defaulting to local. Numeric values are read as Unix seconds, milliseconds, microseconds, or nanoseconds, inferred from magnitude. Fractional parts are preserved.

**Durations.** Unit resolves in order: `--duration-unit`, then a unit in the value (`"43ms"`), then the key name (`duration_ms` → ms, `request_time` → s), then milliseconds. Values may be numbers, numeric strings, or Go duration strings. Falling back to inference on an ambiguous key like a bare `duration` warns once on stderr.

**Missing fields** are marked unset, not zeroed. A `null` or absent key means no value, so a log without timing reports no latency statistics rather than a misleading `p99: 0ms`. Fields present but unparseable are counted separately and summarised at the end of the run.

### nginx combined

```
203.0.113.42 - - [10/Oct/2024:14:03:11 +0300] "GET /api/products?page=2 HTTP/1.1" 200 4821 "https://shop.example/" "Mozilla/5.0"
```

The standard combined format carries **no request duration**, so latency statistics are unavailable. To include timing, append `$request_time`:

```nginx
log_format combined_timed '$remote_addr - $remote_user [$time_local] '
                          '"$request" $status $body_bytes_sent '
                          '"$http_referer" "$http_user_agent" $request_time';

access_log /var/log/nginx/access.log combined_timed;
```

`loglens` detects the extra field automatically.

**Durations.** No key name to infer from, so the unit is `--duration-unit` if given, otherwise **seconds**, matching nginx. Pass the flag explicitly if your `log_format` emits milliseconds. Getting it wrong is a 1000x error that still looks plausible.

### Malformed lines
Counted and reported, never silently dropped. The first 10 print to stderr with line numbers; the rest are summarised. `--strict` exits non-zero when the malformed rate exceeds 1%.
