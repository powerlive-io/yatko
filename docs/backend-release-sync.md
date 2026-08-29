# Backend release synchronization

The continuously running Go backend polls GitHub and mirrors missing release
assets to Alibaba Cloud OSS. GitHub remains the metadata source. A
`_manifest.json` object marks a release as complete; until that object is
reachable through the CDN, download links continue to use GitHub.

No GitHub Actions or cross-repository triggers are required.

## Alibaba Cloud OSS

Create an OSS bucket and a RAM user that can read, write, inspect, and delete
objects under `releases/*`. Configure Alibaba Cloud CDN with that bucket as its
origin and connect the intended CDN hostname.

Configure these variables on the Coolify backend service:

```text
OSS_ENDPOINT=https://oss-cn-hangzhou.aliyuncs.com
OSS_BUCKET=your-bucket-name
OSS_ACCESS_KEY_ID=...
OSS_ACCESS_KEY_SECRET=...
RELEASE_CDN_URL=https://cdn.powerlive.io
ALLOWED_REPOS=Xinrea/bili-shadowreplay,Xinrea/JLiverTool,Xinrea/JoiAsk
RELEASE_SYNC_INTERVAL=10m
RELEASE_SYNC_LIMIT=5
GITHUB_TOKEN=...
TRUSTED_PROXIES=172.16.0.0/12
```

`GITHUB_TOKEN` is optional for public repositories but avoids GitHub's low
anonymous API limit. `RELEASE_SYNC_INTERVAL` has a minimum of one minute, and
`RELEASE_SYNC_LIMIT` controls how many recent releases are checked per
repository. Set `TRUSTED_PROXIES` to the actual Coolify/Docker proxy network so
per-client rate limiting can safely use `X-Forwarded-For`; do not use
`0.0.0.0/0`.

If the Coolify server is an Alibaba Cloud server in the same region as the
bucket, use the region's internal OSS endpoint to avoid public transfer between
the backend and OSS.

The sync worker stays disabled when no OSS variables are present. If any OSS
variable is supplied, all four required OSS variables must be supplied or the
backend refuses to start. Do not set `RELEASE_CDN_URL` until the custom domain
can serve `/releases/.../_manifest.json`.

Configure the frontend service with the backend's private Coolify URL:

```text
BACKEND_URL=http://powerlive-backend:8080
```

Expose the frontend as `https://app.powerlive.io`. Only the frontend needs a
public route; `/api`, `/dl`, `/code`, and `/health` must be forwarded to the Go
backend by the deployment proxy.

## Runtime behavior

The worker checks releases once at backend startup and then at the configured
interval. Existing completion manifests make polling inexpensive. Assets stream
from GitHub into OSS without being buffered in memory or written to local disk.
An interrupted or size-mismatched upload does not produce a completion
manifest, so the next poll retries it.

Run one backend replica while using the built-in poller. The completion
manifest makes retries idempotent, but separate replicas can still copy the
same newly published asset concurrently.

Objects use this stable key layout:

```text
releases/{owner}/{repo}/{tag}/{asset-name}
releases/{owner}/{repo}/{tag}/_manifest.json
```

Release assets receive a one-year immutable cache header. Avoid replacing an
asset under an already published tag; publish a new tag instead.
