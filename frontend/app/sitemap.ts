import type { MetadataRoute } from "next";
import { SITE_URL } from "@/lib/site";
import { PROJECTS } from "@/lib/projects";

/** Fixed project pages are part of this portal's crawlable public surface. */
export default function sitemap(): MetadataRoute.Sitemap {
  const lastModified = new Date();
  return [
    {
      url: SITE_URL,
      lastModified,
      changeFrequency: "weekly",
      priority: 1,
    },
    {
      url: `${SITE_URL}/privacy`,
      lastModified,
      changeFrequency: "yearly",
      priority: 0.3,
    },
    ...PROJECTS.map(({ owner, repo }) => ({
      url: `${SITE_URL}/${owner}/${repo}`,
      lastModified,
      changeFrequency: "daily" as const,
      priority: 0.8,
    })),
  ];
}
