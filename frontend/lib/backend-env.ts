/** Shared frontend → Go backend connection settings. */
export const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

/** Cap wall time so hung upstreams cannot pin a Fluid instance near max duration. */
export const BACKEND_FETCH_TIMEOUT_MS = 8_000;

/**
 * How long Next may reuse a cached backend response before revalidating.
 * Keep this short enough for asynchronously mirrored releases to switch from
 * their GitHub fallback URLs to CDN URLs promptly.
 */
export const BACKEND_FETCH_REVALIDATE_SECONDS = 60;
