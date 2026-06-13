// The application-stage vocabulary, in pipeline order (active stages then
// terminal). SOURCE OF TRUTH is the Go `validStages` set in
// internal/handler/user_jobs.go — keep this in sync when it changes. Drift is not
// fatal: humanizeStage renders an unknown value as a readable label.

export interface StageOption {
  value: string;
  label: string;
}

export const STAGES: StageOption[] = [
  { value: 'applied', label: 'Applied' },
  { value: 'screening', label: 'Screening' },
  { value: 'responded', label: 'Responded' },
  { value: 'interview', label: 'Interview' },
  { value: 'offer', label: 'Offer' },
  { value: 'accepted', label: 'Accepted' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'withdrawn', label: 'Withdrawn' },
];

const LABELS = new Map(STAGES.map((s) => [s.value, s.label]));

/** A human label for a stage value; the value itself (title-cased fallback) when
 *  not in the known vocabulary. */
export function humanizeStage(stage: string): string {
  return LABELS.get(stage) ?? stage.charAt(0).toUpperCase() + stage.slice(1);
}
