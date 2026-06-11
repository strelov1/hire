import { cn } from '$lib/utils';

// The shared three-state look of a selectable facet pill: idle (secondary fill),
// selected-include (primary fill), selected-exclude (muted destructive,
// struck through). Callers add their own size classes via `extra`.
export function pillClass(active: boolean, exclude: boolean, extra = ''): string {
  return cn(
    'rounded-full border font-medium transition-colors active:translate-y-px',
    !active && 'border-transparent bg-secondary text-secondary-foreground hover:bg-accent',
    active && !exclude && 'border-transparent bg-primary text-primary-foreground',
    active && exclude && 'border-destructive/30 bg-destructive/15 text-destructive line-through',
    extra,
  );
}
