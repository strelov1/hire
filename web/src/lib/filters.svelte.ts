// Job search filters: the model, its URL <-> state (de)serialization, and a
// reactive store that mirrors the filters into the URL query so they survive
// reloads, sharing, and back/forward. Param names match what the search API
// (GET /api/v1/jobs/search) expects, including the `<param>_exclude` and
// `<param>_mode=and` conventions.

import { router } from './router.svelte';
import { FACETS } from './facets';

/** One facet's selection: the chosen values, whether it filters by inclusion or
 *  exclusion, and (for facets that allow it) whether selected values are ANDed
 *  (match all) instead of ORed (match any). */
export interface FacetState {
  values: string[];
  exclude: boolean;
  matchAll: boolean;
}

export interface JobFilters {
  q: string;
  /** Facet state keyed by the facet's query param (see FACETS). */
  facets: Record<string, FacetState>;
  visa: boolean;
  salaryMin: number | null;
}

function emptyFacet(): FacetState {
  return { values: [], exclude: false, matchAll: false };
}

function emptyFacets(): Record<string, FacetState> {
  const out: Record<string, FacetState> = {};
  for (const f of FACETS) out[f.param] = emptyFacet();
  return out;
}

export function emptyFilters(): JobFilters {
  return { q: '', facets: emptyFacets(), visa: false, salaryMin: null };
}

/** Serialize filters to URL query params (the shape the search API reads). */
export function filtersToParams(f: JobFilters): URLSearchParams {
  const p = new URLSearchParams();
  if (f.q) p.set('q', f.q);
  for (const def of FACETS) {
    const st = f.facets[def.param];
    if (!st || st.values.length === 0) continue;
    const key = st.exclude ? `${def.param}_exclude` : def.param;
    for (const v of st.values) p.append(key, v);
    // AND-mode is per facet and only meaningful with more than one included value.
    if (st.matchAll && !st.exclude && st.values.length > 1) {
      p.set(`${def.param}_mode`, 'and');
    }
  }
  if (f.visa) p.set('visa_sponsorship', 'true');
  if (f.salaryMin != null) p.set('salary_min', String(f.salaryMin));
  return p;
}

/** Parse filters back from URL query params. Exclude takes precedence over
 *  include when both appear for the same facet. */
export function filtersFromParams(p: URLSearchParams): JobFilters {
  const f = emptyFilters();
  f.q = p.get('q') ?? '';
  for (const def of FACETS) {
    const exclude = p.getAll(`${def.param}_exclude`);
    const include = p.getAll(def.param);
    const matchAll = p.get(`${def.param}_mode`) === 'and';
    if (exclude.length > 0) f.facets[def.param] = { values: exclude, exclude: true, matchAll };
    else if (include.length > 0) f.facets[def.param] = { values: include, exclude: false, matchAll };
  }
  f.visa = p.get('visa_sponsorship') === 'true';
  const salary = Number(p.get('salary_min'));
  f.salaryMin = p.get('salary_min') && !Number.isNaN(salary) ? salary : null;
  return f;
}

/** Total selected facet values (plus visa/salary) — drives the mobile badge. */
export function activeFilterCount(f: JobFilters): number {
  let n = 0;
  for (const def of FACETS) n += f.facets[def.param]?.values.length ?? 0;
  if (f.visa) n += 1;
  if (f.salaryMin != null) n += 1;
  return n;
}

/** Reactive filter state mirrored into the URL. Owned by the jobs view; all
 *  mutations go through its methods so every change updates state and URL. */
export class FilterStore {
  value = $state<JobFilters>(emptyFilters());

  constructor() {
    this.value = filtersFromParams(router.query);
  }

  get active(): number {
    return activeFilterCount(this.value);
  }

  facet(param: string): FacetState {
    return this.value.facets[param] ?? emptyFacet();
  }

  setQuery(q: string) {
    this.value = { ...this.value, q };
    this.#commit();
  }

  setVisa(on: boolean) {
    this.value = { ...this.value, visa: on };
    this.#commit();
  }

  setSalaryMin(n: number | null) {
    this.value = { ...this.value, salaryMin: n };
    this.#commit();
  }

  /** Toggle a facet between match-all (AND) and match-any (OR) of its values. */
  setMatchAll(param: string, on: boolean) {
    this.#setFacet(param, { ...this.facet(param), matchAll: on });
  }

  /** Add the value to a facet if absent, remove it if present (pills). */
  toggle(param: string, v: string) {
    const st = this.facet(param);
    const has = st.values.includes(v);
    this.#setFacet(param, { ...st, values: has ? st.values.filter((x) => x !== v) : [...st.values, v] });
  }

  /** Add a token to a facet (token inputs); no-op on blank or duplicate. */
  add(param: string, raw: string) {
    const v = raw.trim();
    const st = this.facet(param);
    if (!v || st.values.includes(v)) return;
    this.#setFacet(param, { ...st, values: [...st.values, v] });
  }

  remove(param: string, v: string) {
    const st = this.facet(param);
    this.#setFacet(param, { ...st, values: st.values.filter((x) => x !== v) });
  }

  /** Reset a single facet (values + exclude mode) — the per-section clear. */
  clearFacet(param: string) {
    this.#setFacet(param, emptyFacet());
  }

  /** Switch a facet between include and exclude mode (the "Исключить" link). */
  setExclude(param: string, exclude: boolean) {
    this.#setFacet(param, { ...this.facet(param), exclude });
  }

  clear() {
    this.value = emptyFilters();
    this.#commit();
  }

  /** Re-read filters from the URL (browser back/forward). No-op when already in
   *  sync, which also breaks the write-back loop after our own setQuery. */
  syncFromUrl() {
    if (router.query.toString() === filtersToParams(this.value).toString()) return;
    this.value = filtersFromParams(router.query);
  }

  #setFacet(param: string, st: FacetState) {
    this.value = { ...this.value, facets: { ...this.value.facets, [param]: st } };
    this.#commit();
  }

  #commit() {
    router.setQuery(filtersToParams(this.value));
  }
}
