import { ImageResponse } from "next/og";
import { outfitFontOption } from "@/lib/og-font";
import { SITE_DOMAIN, SITE_NAME, SITE_TAGLINE } from "@/lib/site";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = `${SITE_NAME} — ${SITE_TAGLINE}`;

export default async function Image() {
  const title = SITE_NAME;
  const tagline = "为你的平台选择合适的最新版本";

  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          background: "#09090b",
          color: "#fafafa",
          fontFamily: "Outfit",
          padding: "0 80px",
        }}
      >
        <div style={{ display: "flex", fontSize: 28, color: "#a1a1aa", marginBottom: 28 }}>
          {SITE_DOMAIN}
        </div>
        <div style={{ display: "flex", fontSize: 96, fontWeight: 600, letterSpacing: "-0.04em" }}>
          {title}
        </div>
        <div
          style={{
            display: "flex",
            fontSize: 28,
            color: "#a1a1aa",
            marginTop: 28,
            maxWidth: 900,
            textAlign: "center",
          }}
        >
          {tagline}
        </div>
      </div>
    ),
    {
      ...size,
      fonts: [outfitFontOption],
    },
  );
}
