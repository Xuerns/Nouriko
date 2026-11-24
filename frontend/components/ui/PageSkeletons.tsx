import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

interface SkeletonStatusProps {
  label?: string;
}

export function ChatPageSkeleton({ label = "Loading..." }: SkeletonStatusProps) {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label={label}
      className="flex h-full min-h-0 flex-col gap-4"
    >
      <span className="sr-only">{label}</span>

      <div className="shrink-0 flex items-center justify-between gap-3 border-b border-border pb-3.5">
        <div className="space-y-2">
          <Skeleton className="h-4 w-28 rounded-md" />
          <Skeleton className="h-3 w-40 rounded-md" />
        </div>
        <Skeleton className="h-9 w-24 rounded-full" />
      </div>

      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 px-2 py-8">
        <Skeleton className="size-12 rounded-full" />
        <Skeleton className="h-6 w-52 rounded-lg" />
        <Skeleton className="h-4 w-72 max-w-full rounded-md" />
      </div>

      <div className="shrink-0 space-y-2.5">
        <div className="flex flex-wrap gap-2">
          <Skeleton className="h-9 w-44 rounded-full" />
          <Skeleton className="h-9 w-36 rounded-full" />
          <Skeleton className="h-9 w-40 rounded-full" />
        </div>
        <div className="flex items-center gap-2 rounded-3xl border border-border bg-card px-3 py-2.5">
          <Skeleton className="size-9 shrink-0 rounded-full" />
          <Skeleton className="h-5 flex-1 rounded-md" />
          <Skeleton className="size-9 shrink-0 rounded-full" />
        </div>
      </div>
    </div>
  );
}

export function SummaryPageSkeleton({
  label = "Loading...",
}: SkeletonStatusProps) {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label={label}
      className="space-y-6"
    >
      <span className="sr-only">{label}</span>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-2">
          <Skeleton className="h-6 w-44 rounded-lg" />
          <Skeleton className="h-4 w-56 rounded-md" />
        </div>
        <Skeleton className="h-10 w-44 rounded-full" />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
        <div className="space-y-5 lg:col-span-5">
          <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5">
            <div className="space-y-3">
              <div className="flex items-center justify-between gap-2">
                <Skeleton className="h-4 w-28 rounded-md" />
                <Skeleton className="h-6 w-16 rounded-full" />
              </div>
              <Skeleton className="h-10 w-40 rounded-lg" />
              <Skeleton className="h-2.5 w-full rounded-full" />
              <Skeleton className="h-11 w-full rounded-full" />
            </div>

            <Separator className="bg-border" />

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Skeleton className="h-3.5 w-24 rounded-md" />
                <Skeleton className="h-6 w-28 rounded-md" />
                <Skeleton className="h-3 w-16 rounded-md" />
              </div>
              <div className="space-y-2">
                <Skeleton className="h-3.5 w-24 rounded-md" />
                <Skeleton className="h-6 w-20 rounded-md" />
                <Skeleton className="h-3 w-20 rounded-md" />
              </div>
            </div>
          </Card>

          <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-4">
            <Skeleton className="h-5 w-36 rounded-md" />
            <div className="space-y-4">
              {Array.from({ length: 4 }).map((_, index) => (
                <div key={index} className="space-y-2">
                  <div className="flex items-center justify-between gap-2">
                    <Skeleton className="h-3.5 w-20 rounded-md" />
                    <Skeleton className="h-3.5 w-24 rounded-md" />
                  </div>
                  <Skeleton className="h-2 w-full rounded-full" />
                </div>
              ))}
            </div>
          </Card>
        </div>

        <div className="lg:col-span-7">
          <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-4 sm:p-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <Skeleton className="h-5 w-36 rounded-md" />
              <div className="flex gap-1.5">
                <Skeleton className="h-8 w-16 rounded-full" />
                <Skeleton className="h-8 w-16 rounded-full" />
                <Skeleton className="h-8 w-16 rounded-full" />
              </div>
            </div>

            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, index) => (
                <div
                  key={index}
                  className="space-y-3 rounded-2xl border border-border bg-background p-4"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="space-y-2">
                      <Skeleton className="h-4 w-44 rounded-md" />
                      <Skeleton className="h-3 w-28 rounded-md" />
                    </div>
                    <Skeleton className="h-6 w-20 rounded-full" />
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Skeleton className="h-6 w-16 rounded-full" />
                    <Skeleton className="h-6 w-16 rounded-full" />
                    <Skeleton className="h-6 w-16 rounded-full" />
                    <Skeleton className="h-6 w-16 rounded-full" />
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}

const CHART_BAR_HEIGHTS = [
  "h-16",
  "h-24",
  "h-20",
  "h-28",
  "h-14",
  "h-24",
  "h-20",
] as const;

export function HistoryContentSkeleton({
  label = "Loading...",
}: SkeletonStatusProps) {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label={label}
      className="space-y-6"
    >
      <span className="sr-only">{label}</span>

      <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-5 sm:p-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="space-y-2">
            <Skeleton className="h-5 w-44 rounded-md" />
            <Skeleton className="h-3.5 w-60 rounded-md" />
          </div>
          <Skeleton className="h-7 w-24 rounded-full" />
        </div>

        <div className="flex h-44 items-end justify-between gap-3 px-2 pt-4">
          {CHART_BAR_HEIGHTS.map((heightClass, index) => (
            <div
              key={index}
              className="flex flex-1 flex-col items-center gap-2"
            >
              <Skeleton className={`w-full max-w-10 rounded-lg ${heightClass}`} />
              <Skeleton className="h-3 w-8 rounded-md" />
            </div>
          ))}
        </div>
      </Card>

      <div className="space-y-3.5">
        <div className="flex flex-wrap items-center justify-between gap-3 px-1">
          <Skeleton className="h-5 w-36 rounded-md" />
          <div className="flex gap-2">
            <Skeleton className="h-10 w-20 rounded-full" />
            <Skeleton className="h-10 w-20 rounded-full" />
            <Skeleton className="h-10 w-20 rounded-full" />
          </div>
        </div>

        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <Card
              key={index}
              className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0"
            >
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="space-y-2">
                  <Skeleton className="h-4 w-40 rounded-md" />
                  <Skeleton className="h-3.5 w-52 rounded-md" />
                </div>
                <div className="flex items-center gap-2">
                  <Skeleton className="h-7 w-20 rounded-full" />
                  <Skeleton className="size-8 rounded-full" />
                </div>
              </div>
            </Card>
          ))}
        </div>
      </div>
    </div>
  );
}

export function HistoryPageSkeleton({
  label = "Loading...",
}: SkeletonStatusProps) {
  return (
    <div className="space-y-6">
      <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 sm:p-6">
        <div className="space-y-2">
          <Skeleton className="h-7 w-40 rounded-lg" />
          <Skeleton className="h-4 w-64 max-w-full rounded-md" />
        </div>
      </Card>
      <HistoryContentSkeleton label={label} />
    </div>
  );
}

export function ProfilePageSkeleton({
  label = "Loading...",
}: SkeletonStatusProps) {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label={label}
      className="space-y-5"
    >
      <span className="sr-only">{label}</span>

      <div className="space-y-2 px-0.5">
        <Skeleton className="h-7 w-36 rounded-lg" />
        <Skeleton className="h-4 w-64 max-w-full rounded-md" />
      </div>

      <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <Skeleton className="size-12 shrink-0 rounded-full" />
            <div className="space-y-2">
              <Skeleton className="h-4 w-32 rounded-md" />
              <Skeleton className="h-3.5 w-44 rounded-md" />
            </div>
          </div>
          <div className="flex gap-1.5">
            <Skeleton className="h-7 w-32 rounded-full" />
            <Skeleton className="h-7 w-20 rounded-full" />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
          {Array.from({ length: 5 }).map((_, index) => (
            <div
              key={index}
              className={`space-y-2 rounded-2xl bg-muted/60 px-3 py-2.5 ${
                index === 0 ? "col-span-2 sm:col-span-1" : ""
              }`}
            >
              <Skeleton className="h-3 w-16 rounded-md" />
              <Skeleton className="h-4 w-20 rounded-md" />
            </div>
          ))}
        </div>
      </Card>

      <div className="flex w-fit gap-1.5 rounded-full border border-border bg-card p-1">
        <Skeleton className="h-9 w-28 rounded-full" />
        <Skeleton className="h-9 w-24 rounded-full" />
        <Skeleton className="h-9 w-24 rounded-full" />
      </div>

      <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-5">
        <div className="space-y-2">
          <Skeleton className="h-5 w-32 rounded-md" />
          <Skeleton className="h-3.5 w-80 max-w-full rounded-md" />
        </div>

        <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-12">
          <div className="space-y-1.5 sm:col-span-7">
            <Skeleton className="h-3.5 w-24 rounded-md" />
            <Skeleton className="h-10 w-full rounded-2xl" />
          </div>
          <div className="space-y-1.5 sm:col-span-5">
            <Skeleton className="h-3.5 w-20 rounded-md" />
            <Skeleton className="h-10 w-full rounded-2xl" />
          </div>
        </div>

        <div className="grid grid-cols-3 gap-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <div key={index} className="space-y-1.5">
              <Skeleton className="h-3.5 w-14 rounded-md" />
              <Skeleton className="h-10 w-full rounded-2xl" />
            </div>
          ))}
        </div>

        <div className="space-y-2">
          <Skeleton className="h-3.5 w-28 rounded-md" />
          <div className="grid grid-cols-3 gap-2">
            <Skeleton className="h-12 w-full rounded-[12px]" />
            <Skeleton className="h-12 w-full rounded-[12px]" />
            <Skeleton className="h-12 w-full rounded-[12px]" />
          </div>
        </div>
      </Card>
    </div>
  );
}

export function OnboardingPageSkeleton({
  label = "Loading...",
}: SkeletonStatusProps) {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label={label}
      className="grid grid-cols-1 gap-8 lg:grid-cols-12 lg:items-start"
    >
      <span className="sr-only">{label}</span>

      <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-6 sm:p-8 lg:col-span-7">
        <div className="flex items-center justify-between gap-3">
          <Skeleton className="h-7 w-48 rounded-lg" />
          <Skeleton className="h-7 w-14 rounded-full" />
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Skeleton className="h-4 w-28 rounded-md" />
            <Skeleton className="h-11 w-full rounded-2xl" />
          </div>
          <div className="space-y-2">
            <Skeleton className="h-4 w-24 rounded-md" />
            <div className="grid grid-cols-2 gap-2">
              <Skeleton className="h-11 w-full rounded-2xl" />
              <Skeleton className="h-11 w-full rounded-2xl" />
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <div key={index} className="space-y-2">
              <Skeleton className="h-4 w-20 rounded-md" />
              <Skeleton className="h-11 w-full rounded-2xl" />
            </div>
          ))}
        </div>

        <div className="space-y-2">
          <Skeleton className="h-4 w-32 rounded-md" />
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-3">
            <Skeleton className="h-20 w-full rounded-2xl" />
            <Skeleton className="h-20 w-full rounded-2xl" />
            <Skeleton className="h-20 w-full rounded-2xl" />
          </div>
        </div>

        <Skeleton className="h-12 w-full rounded-full" />
      </Card>

      <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5 lg:col-span-5">
        <div className="flex items-center gap-3.5">
          <Skeleton className="size-12 shrink-0 rounded-full" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-32 rounded-md" />
            <Skeleton className="h-3.5 w-24 rounded-md" />
          </div>
        </div>
        <Skeleton className="h-24 w-full rounded-2xl" />
        <Skeleton className="h-2.5 w-full rounded-full" />
        <div className="grid grid-cols-2 gap-2.5">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-20 w-full rounded-2xl" />
          ))}
        </div>
      </Card>
    </div>
  );
}

export function AuthSplitSkeleton({
  label = "Loading...",
}: SkeletonStatusProps) {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label={label}
      className="grid grid-cols-1 items-center gap-10 py-4 lg:grid-cols-12 lg:gap-14"
    >
      <span className="sr-only">{label}</span>

      <div className="space-y-5 lg:col-span-7">
        <div className="space-y-3">
          <Skeleton className="h-10 w-80 max-w-full rounded-lg" />
          <Skeleton className="h-10 w-64 max-w-full rounded-lg" />
          <Skeleton className="h-4 w-96 max-w-full rounded-md" />
        </div>
        <div className="space-y-2.5 pt-2">
          <Skeleton className="h-5 w-60 rounded-md" />
          <Skeleton className="h-5 w-52 rounded-md" />
          <Skeleton className="h-5 w-56 rounded-md" />
        </div>
      </div>

      <Card className="w-full rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5 sm:p-8 lg:col-span-5">
        <div className="space-y-2">
          <Skeleton className="h-6 w-36 rounded-lg" />
          <Skeleton className="h-4 w-56 rounded-md" />
        </div>
        <div className="space-y-4">
          <div className="space-y-1.5">
            <Skeleton className="h-3.5 w-16 rounded-md" />
            <Skeleton className="h-11 w-full rounded-2xl" />
          </div>
          <div className="space-y-1.5">
            <Skeleton className="h-3.5 w-20 rounded-md" />
            <Skeleton className="h-11 w-full rounded-2xl" />
          </div>
          <Skeleton className="h-11 w-full rounded-full" />
        </div>
      </Card>
    </div>
  );
}

export function AuthCardSkeleton({
  label = "Loading...",
}: SkeletonStatusProps) {
  return (
    <Card
      role="status"
      aria-busy="true"
      aria-label={label}
      className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5"
    >
      <span className="sr-only">{label}</span>
      <div className="space-y-2">
        <Skeleton className="h-6 w-44 rounded-lg" />
        <Skeleton className="h-4 w-full rounded-md" />
        <Skeleton className="h-4 w-4/5 rounded-md" />
      </div>
      <div className="space-y-3">
        <Skeleton className="h-11 w-full rounded-2xl" />
        <Skeleton className="h-11 w-full rounded-full" />
      </div>
    </Card>
  );
}
