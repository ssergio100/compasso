import { CircleAlert, Inbox, LoaderCircle, RotateCcw, type LucideIcon } from "lucide-react";
import { Button } from "./Button";
import { Surface } from "./Surface";

type PageStateKind = "loading" | "error" | "empty";

const icons: Record<PageStateKind, LucideIcon> = {
  loading: LoaderCircle,
  error: CircleAlert,
  empty: Inbox,
};

export function PageState({
  actionLabel,
  description,
  kind,
  onAction,
  title,
  headingLevel = "h1",
}: {
  actionLabel?: string;
  description: string;
  kind: PageStateKind;
  onAction?: () => void;
  title: string;
  headingLevel?: "h1" | "h2";
}) {
  const Icon = icons[kind];
  const Heading = headingLevel;
  return (
    <Surface className="grid min-h-[18rem] place-items-center p-6 text-center" role={kind === "error" ? "alert" : "status"}>
      <div className="max-w-sm">
        <span className={`mx-auto grid size-12 place-items-center rounded-2xl ${kind === "error" ? "bg-danger-soft text-danger" : "bg-brand-soft text-brand"}`}>
          <Icon aria-hidden="true" size={23} />
        </span>
        <Heading className="mt-4 text-lg font-extrabold">{title}</Heading>
        <p className="mt-2 text-sm leading-relaxed text-muted">{description}</p>
        {actionLabel && onAction ? (
          <Button className="mt-5" leadingIcon={kind === "error" ? <RotateCcw aria-hidden="true" size={17} /> : undefined} onClick={onAction}>
            {actionLabel}
          </Button>
        ) : null}
      </div>
    </Surface>
  );
}
