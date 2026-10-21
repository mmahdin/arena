import { cn } from "@/lib/cn";
import "./Skeleton.css";

export function Skeleton({
  className,
  style,
}: {
  className?: string;
  style?: React.CSSProperties;
}) {
  return <div className={cn("skeleton", className)} style={style} aria-hidden="true" />;
}

export function PageSkeleton() {
  return (
    <div className="page-skeleton" aria-busy="true" aria-label="در حال بارگذاری">
      <Skeleton className="page-skeleton__title" />
      <Skeleton className="page-skeleton__hero" />
      <div className="page-skeleton__row">
        <Skeleton className="page-skeleton__block" />
        <Skeleton className="page-skeleton__block" />
      </div>
      <Skeleton className="page-skeleton__line" />
      <Skeleton className="page-skeleton__line page-skeleton__line--short" />
    </div>
  );
}
