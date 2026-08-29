"use client";

import { useState } from "react";
import Link from "next/link";
import { PROJECTS } from "@/lib/projects";
import { ReleaseLoading } from "./p/[owner]/[repo]/release-loading";

export function ProjectGrid() {
  const [navigating, setNavigating] = useState(false);

  if (navigating) return <ReleaseLoading />;

  return (
    <div className="grid gap-3 text-left">
      {PROJECTS.map((project) => (
        <Link
          key={`${project.owner}/${project.repo}`}
          href={`/${project.owner}/${project.repo}`}
          onClick={() => setNavigating(true)}
          className="group rounded-2xl border border-border bg-surface/50 px-5 py-4 text-left transition-[background-color,border-color,transform] hover:border-foreground/20 hover:bg-foreground/[0.04] active:scale-[0.99]"
        >
          <span className="flex items-center justify-between gap-4">
            <span>
              <span className="block text-lg font-semibold tracking-tight">
                {project.name}
              </span>
              <span className="mt-1 block text-sm text-muted">{project.description}</span>
            </span>
            <ArrowIcon />
          </span>
        </Link>
      ))}
    </div>
  );
}

function ArrowIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 20 20"
      fill="none"
      aria-hidden
      className="shrink-0 text-muted transition-transform group-hover:translate-x-0.5"
    >
      <path
        d="m8 5 5 5-5 5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
