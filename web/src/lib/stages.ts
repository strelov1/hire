// The application-stage vocabulary, in pipeline order (active stages then
// terminal). STAGE_VALUES is generated from the Go userjob.Stages slice in
// internal/userjob via cmd/gen-contracts. Drift is not fatal: humanizeStage
// renders an unknown value as a readable label.

import { STAGE_VALUES } from './generated/contracts';

export interface StageOption {
  value: string;
  label: string;
}

const STAGE_LABELS: Record<string, string> = {
  applied: 'Applied',
  screening: 'Screening',
  responded: 'Responded',
  interview: 'Interview',
  offer: 'Offer',
  accepted: 'Accepted',
  rejected: 'Rejected',
  withdrawn: 'Withdrawn',
};

/** A human label for a stage value; the value itself (title-cased fallback) when
 *  not in the known vocabulary. */
export function humanizeStage(stage: string): string {
  return STAGE_LABELS[stage] ?? stage.charAt(0).toUpperCase() + stage.slice(1);
}

export const STAGES: StageOption[] = STAGE_VALUES.map((value) => ({
  value,
  label: STAGE_LABELS[value] ?? humanizeStage(value),
}));
