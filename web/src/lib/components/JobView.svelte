<script lang="ts">
  import { browser } from '$app/environment';
  import { ArrowRight, Bookmark, Check } from '@lucide/svelte';
  import { markJobApplied, recordJobView, saveJob, unsaveJob } from '$lib/api';
  import { authStore } from '$lib/auth.svelte';
  import { formatSalary, summaryFacets } from '$lib/enrichment';
  import type { Job, UserJob } from '$lib/types';
  import { Badge, Button } from '$lib/ui';
  import { formatDate } from '$lib/utils';
  import CompanyLogo from './CompanyLogo.svelte';

  // The job is server-rendered: it arrives as a prop from the route's `load`, so
  // the article's content is in the initial HTML. Only the per-user interactions
  // below hydrate client-side.
  let { job }: { job: Job } = $props();

  // The signed-in user's interaction with this job (null when signed out or not
  // yet loaded). `showApplyPrompt` is the post-click "Did you apply?" question.
  let interaction = $state.raw<UserJob | null>(null);
  let showApplyPrompt = $state(false);
  const applied = $derived(interaction?.applied_at != null);
  const saved = $derived(interaction?.saved_at != null);

  // Presentational values derived from the (server-rendered) job.
  const posted = $derived(formatDate(job.posted_at));
  const e = $derived(job.enrichment ?? {});
  const salary = $derived(formatSalary(e));
  const facets = $derived(summaryFacets(job));

  // Record a view for signed-in users once the page hydrates (browser only —
  // authStore is client-side). Silent history that also tells us whether they
  // already applied; a failed view must not break the page. Re-runs on client
  // navigation to another job, resetting the per-user state first.
  $effect(() => {
    const slug = job.public_slug; // track the current job
    interaction = null;
    showApplyPrompt = false;
    if (!browser || !authStore.isAuthenticated) return;
    recordJobView(slug)
      .then((rec) => {
        if (job.public_slug === slug) interaction = rec;
      })
      .catch(() => {});
  });

  // The Apply link opens the external posting; once the user has gone to apply,
  // offer the "Did you apply?" choice (only when signed in and not already applied).
  function onApplyClick() {
    if (authStore.isAuthenticated && !applied) showApplyPrompt = true;
  }

  async function confirmApplied() {
    try {
      interaction = await markJobApplied(job.public_slug);
    } catch {
      // Leave the prompt up so the user can retry; nothing else to do.
      return;
    }
    showApplyPrompt = false;
  }

  // "No": purely local — the job must not enter the tracker.
  function dismissApplyPrompt() {
    showApplyPrompt = false;
  }

  // The toggle flips on the server's answer, not optimistically: both endpoints
  // return the full interaction, so the button can never drift from the truth.
  async function toggleSave() {
    try {
      interaction = saved ? await unsaveJob(job.public_slug) : await saveJob(job.public_slug);
    } catch {
      // Leave the current state; the user can retry.
    }
  }
</script>

<article class="flex flex-col gap-6">
  <header class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div class="flex items-start gap-3">
        <CompanyLogo name={job.company} size="size-10" />
        <div class="flex flex-col gap-1">
          <p class="text-sm text-muted-foreground">
            {#if job.company_slug}
              <a href={`/companies/${job.company_slug}`} class="hover:text-foreground hover:underline">
                {job.company || 'Unknown company'}
              </a>
            {:else}
              {job.company || 'Unknown company'}
            {/if}
            {#if job.location}· {job.location}{/if}
          </p>
          <h1 class="text-2xl font-semibold tracking-tight">{job.title}</h1>
        </div>
      </div>

      <div class="flex shrink-0 flex-col items-end gap-2">
        {#if !job.closed_at}
          <Button
            variant="primary"
            href={job.url}
            target="_blank"
            rel="noopener noreferrer"
            onclick={onApplyClick}
          >
            Show <ArrowRight class="size-4" />
          </Button>
        {/if}
        {#if authStore.isAuthenticated}
          <Button variant="outline" size="sm" onclick={toggleSave} aria-pressed={saved}>
            <Bookmark class={saved ? 'size-4 fill-current' : 'size-4'} />
            {saved ? 'Saved' : 'Save'}
          </Button>
        {/if}
        {#if applied}
          <Badge variant="secondary"><Check class="mr-1 size-3.5" /> You applied</Badge>
        {/if}
      </div>
    </div>

    {#if job.closed_at}
      {@const closed = formatDate(job.closed_at)}
      <div class="rounded-md border border-border bg-secondary px-4 py-3 text-sm">
        This position is no longer accepting applications{#if closed}
          (closed {closed}){/if}.
      </div>
    {/if}

    {#if showApplyPrompt && !applied}
      <div
        class="flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-secondary px-4 py-3"
      >
        <span class="text-sm">Did you apply to this job?</span>
        <div class="flex items-center gap-2">
          <Button variant="primary" size="sm" onclick={confirmApplied}>Yes, save</Button>
          <Button variant="ghost" size="sm" onclick={dismissApplyPrompt}>No</Button>
        </div>
      </div>
    {/if}

    {#if salary}
      <p class="text-xl font-semibold tabular-nums tracking-tight">{salary}</p>
    {/if}

    {#if facets.length}
      <dl class="flex flex-wrap gap-x-6 gap-y-2 text-sm">
        {#each facets as facet (facet.label)}
          <div class="flex items-baseline gap-1.5">
            <dt class="text-muted-foreground">{facet.label}</dt>
            <dd class="font-medium">{facet.value}</dd>
          </div>
        {/each}
      </dl>
    {/if}

    {#if e.skills?.length}
      <ul class="flex flex-wrap gap-1.5">
        {#each e.skills as skill}
          <li><Badge variant="secondary">{skill}</Badge></li>
        {/each}
      </ul>
    {/if}
  </header>

  <div class="flex flex-wrap items-center gap-2 border-t border-border pt-4">
    <Badge variant="outline">{job.source}</Badge>
    {#if posted}<span class="text-xs text-muted-foreground">Posted {posted}</span>{/if}
  </div>

  {#if job.description}
    <!-- Description is server-sanitized HTML (see internal/sources), safe to render. -->
    <div class="job-description text-sm leading-relaxed">{@html job.description}</div>
  {/if}
</article>

<style>
  .job-description :global(h1),
  .job-description :global(h2),
  .job-description :global(h3),
  .job-description :global(h4) {
    margin-top: 1.25rem;
    margin-bottom: 0.5rem;
    font-weight: 600;
  }

  .job-description :global(p) {
    margin: 0.5rem 0;
  }

  .job-description :global(ul),
  .job-description :global(ol) {
    margin: 0.5rem 0;
    padding-left: 1.25rem;
  }

  .job-description :global(li) {
    display: list-item;
    list-style: disc outside;
    margin: 0.25rem 0;
  }

  /* ATS boards (e.g. Greenhouse) wrap each <li> in a block <p>; collapse its
     margins so the bullet sits beside the text instead of on its own line. */
  .job-description :global(li) > :global(p) {
    margin: 0;
  }

  .job-description :global(a) {
    text-decoration: underline;
  }

  .job-description :global(b),
  .job-description :global(strong) {
    font-weight: 600;
  }
</style>
