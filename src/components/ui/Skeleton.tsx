import { cn } from '@/lib/cn';

/** Base shimmer block. Compose these to sketch a screen's silhouette. */
export function Skeleton({ className }: { className?: string }) {
  return <div className={cn('animate-pulse rounded-sm bg-line', className)} />;
}

/** Skeleton for a standard list row (leading tile + two text lines). */
export function RowSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn('flex items-center gap-3 px-4 py-3.5', className)}>
      <Skeleton className="size-10 rounded-md" />
      <div className="flex-1 space-y-2">
        <Skeleton className="h-3.5 w-3/5" />
        <Skeleton className="h-3 w-2/5" />
      </div>
    </div>
  );
}

export function ListSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <div className="overflow-hidden rounded-lg border border-line bg-surface">
      {Array.from({ length: rows }).map((_, i) => (
        <RowSkeleton key={i} className={i > 0 ? 'border-t border-line-soft' : undefined} />
      ))}
    </div>
  );
}

export function CardSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn('space-y-3 rounded-lg border border-line bg-surface p-4', className)}>
      <Skeleton className="h-4 w-1/2" />
      <Skeleton className="h-3 w-full" />
      <Skeleton className="h-3 w-4/5" />
    </div>
  );
}
