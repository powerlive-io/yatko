import type { Metadata } from "next";
import { PROJECTS } from "@/lib/projects";
import {
  releasePageMetadata,
  renderReleasePage,
} from "../release-route";

type Props = {
  params: Promise<{ owner: string; repo: string; version?: string[] }>;
};

/** Cache release HTML/RSC for a minute. Platform is detected client-side so we
 *  avoid headers()-forced dynamic rendering. Literal required for Next segment
 *  config static analysis — keep aligned with BACKEND_FETCH_REVALIDATE_SECONDS /
 *  the backend CDN availability cache.
 *
 *  Optional catch-all (not a sibling `[version]` segment) so `/p/o/r` and
 *  `/p/o/r/tag` share one route tree. A nested `[version]` page was tripping
 *  Next's "Could not resolve param value for segment: owner" invariant on
 *  2-segment RSC requests.
 *
 *  `generateStaticParams` is required for App Router ISR on dynamic segments —
 *  `revalidate` alone still leaves the route fully dynamic (`private, no-store`).
 *  Seed homepage examples; unknown owner/repo paths still generate on demand
 *  (`dynamicParams` default true). */
export const revalidate = 60;

export function generateStaticParams() {
  // Latest release only (no version segment). Version pins stay on-demand.
  return PROJECTS.map(({ owner, repo }) => ({ owner, repo }));
}

function versionFromParams(version: string[] | undefined): string | undefined {
  if (!version || version.length === 0) return undefined;
  // Tags may contain `/` (e.g. `pkg/v1.0`); join catch-all segments.
  return version.join("/");
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { owner, repo, version: versionSegments } = await params;
  return releasePageMetadata({
    owner,
    repo,
    version: versionFromParams(versionSegments),
  });
}

export default async function ReleasePage({ params }: Props) {
  const { owner, repo, version: versionSegments } = await params;
  return renderReleasePage({
    owner,
    repo,
    version: versionFromParams(versionSegments),
  });
}
