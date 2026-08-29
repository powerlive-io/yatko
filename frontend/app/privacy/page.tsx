import type { Metadata } from "next";
import { BackToYatko } from "@/components/back-to-yatko";
import { GITHUB_REPO_URL, SITE_NAME } from "@/lib/site";

const title = "Privacy";
const description =
  `Privacy policy for ${SITE_NAME}.`;

export const metadata: Metadata = {
  title,
  description,
  alternates: { canonical: "/privacy" },
  openGraph: {
    title: `Privacy · ${SITE_NAME}`,
    description,
    type: "website",
    siteName: SITE_NAME,
    url: "/privacy",
  },
  twitter: {
    card: "summary_large_image",
    title: `Privacy · ${SITE_NAME}`,
    description,
  },
};

export default function PrivacyPage() {
  return (
    <>
      <BackToYatko />
      <main className="flex-1 px-4 py-16 sm:py-20">
        <article className="mx-auto max-w-xl space-y-8 text-sm leading-relaxed text-foreground/90">
          <div className="space-y-3">
            <h1 className="text-3xl font-semibold tracking-tight text-foreground">Privacy</h1>
          </div>

        <section className="space-y-3">
          <h2 className="text-base font-medium text-foreground">What PowerLive.io is</h2>
          <p>
            PowerLive.io provides download pages for Xinrea&apos;s public projects. It is an
            independent service and is{" "}
            <strong className="font-medium text-foreground">not affiliated with GitHub, Inc.</strong>
          </p>
        </section>

        <section className="space-y-3">
          <h2 className="text-base font-medium text-foreground">What we fetch</h2>
          <p>
            Our servers request public release metadata and README content from GitHub. Release
            assets are copied to Alibaba Cloud OSS and normally delivered through our CDN. GitHub
            may be used as a temporary fallback while a new release is synchronizing.
          </p>
        </section>

        <section className="space-y-3">
          <h2 className="text-base font-medium text-foreground">Caching</h2>
          <p>
            Public release responses and files may be cached to improve download speed. These
            entries contain public project data, not your identity.
          </p>
        </section>

        <section className="space-y-3">
          <h2 className="text-base font-medium text-foreground">Accounts and personal data</h2>
          <p>
            PowerLive.io has no user accounts and does not ask for a login. We do not sell personal data.
            Like most websites, infrastructure logs may briefly include technical details such as
            IP address, User-Agent, and the path requested — used for rate limiting, abuse
            prevention, and keeping the service reliable.
          </p>
        </section>

        <section className="space-y-3">
          <h2 className="text-base font-medium text-foreground">Analytics</h2>
          <p>
            The site uses Vercel Analytics and Speed Insights for aggregate traffic and
            performance. Those products are operated by Vercel under their own privacy terms.
          </p>
        </section>

        <section className="space-y-3">
          <h2 className="text-base font-medium text-foreground">Contact</h2>
          <p>
            Questions or requests: open an issue on{" "}
            <a
              href={GITHUB_REPO_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="font-medium text-foreground underline underline-offset-2 hover:no-underline"
            >
              github.com/Xinrea/yatko
            </a>
            . The project source is available under the MIT License.
          </p>
        </section>
      </article>
    </main>
    </>
  );
}
