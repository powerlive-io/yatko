import { ProjectGrid } from "./project-grid";
import { SITE_DOMAIN } from "@/lib/site";
import { chromeTextButton } from "@/components/chrome";

export default function Home() {
  return (
    <main className="flex-1 flex flex-col items-center justify-center px-4 py-20 min-h-[100dvh]">
      <a
        href="https://powerlive.io/"
        target="_blank"
        rel="noopener noreferrer"
        aria-label="返回 PowerLive 主站（在新标签页打开）"
        className={`fixed top-4 left-4 z-50 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-foreground ${chromeTextButton}`}
      >
        <span aria-hidden="true">←</span>
        返回主站
      </a>
      <div className="w-full max-w-xl space-y-12 text-center">
        <div className="space-y-5">
          <div className="space-y-2">
            <h1 className="text-5xl sm:text-7xl font-semibold tracking-tighter leading-[1.05]">
              PowerLive.io
            </h1>
            <p className="text-sm sm:text-base text-muted font-mono tracking-tight">
              {SITE_DOMAIN}
            </p>
          </div>
          <div className="space-y-2 max-w-md mx-auto tracking-normal">
            <p className="text-base sm:text-lg leading-relaxed">
              直播工具
            </p>
            <p className="text-sm sm:text-base text-muted leading-relaxed">
              为你的平台选择合适的版本，快速获取最新发布。
            </p>
          </div>
        </div>

        <ProjectGrid />
      </div>
    </main>
  );
}
