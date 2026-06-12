<script lang="ts">
  import { ArrowRight, Check } from '@lucide/svelte';
  import { getJob, markJobApplied, recordJobView } from '$lib/api';
  import { authStore } from '$lib/auth.svelte';
  import { formatSalary, summaryFacets } from '$lib/enrichment';
  import type { Job, UserJob } from '$lib/types';
  import { Badge, Button } from '$lib/ui';
  import { formatDate } from '$lib/utils';
  import States from './States.svelte';
  import CompanyLogo from './CompanyLogo.svelte';

  let { slug }: { slug: string } = $props();

  let job = $state.raw<Job | null>(null);
  let status = $state<'loading' | 'error' | 'ready'>('loading');

  // The signed-in user's interaction with this job (null when signed out or not
  // yet loaded). `showApplyPrompt` is the post-click "Did you apply?" question.
  let interaction = $state.raw<UserJob | null>(null);
  let showApplyPrompt = $state(false);
  const applied = $derived(interaction?.applied_at != null);

  // Reload whenever the route slug changes.
  $effect(() => {
    const current = slug;
    status = 'loading';
    job = null;
    interaction = null;
    showApplyPrompt = false;
    getJob(current)
      .then((j) => {
        if (current !== slug) return;
        job = j;
        status = 'ready';
        // Record a view for signed-in users: silent history that also tells us
        // whether they already applied. A failed view must not break the page.
        if (authStore.isAuthenticated) {
          recordJobView(j.public_slug)
            .then((rec) => {
              if (current === slug) interaction = rec;
            })
            .catch(() => {});
        }
      })
      .catch(() => {
        if (current !== slug) return;
        status = 'error';
      });
  });

  // The Apply link opens the external posting; once the user has gone to apply,
  // offer the "Did you apply?" choice (only when signed in and not already applied).
  function onApplyClick() {
    if (authStore.isAuthenticated && !applied) showApplyPrompt = true;
  }

  async function confirmApplied() {
    if (!job) return;
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
</script>

{#if status === 'loading'}
  <States state="loading" rows={3} />
{:else if status === 'error' || !job}
  <States state="error" message="Job not found." />
{:else}
  {@const posted = formatDate(job.posted_at)}
  {@const e = job.enrichment ?? {}}
  {@const salary = formatSalary(e)}
  {@const facets = summaryFacets(e)}
  <article class="flex flex-col gap-6">
    <header class="flex flex-col gap-4">
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-start gap-3">
          <CompanyLogo name={job.company} size="size-10" />
          <div class="flex flex-col gap-1">
            <p class="text-sm text-muted-foreground">
              {#if job.company_slug}
                <a
                  href={`/companies/${job.company_slug}`}
                  class="hover:text-foreground hover:underline"
                >
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
          <Button
            variant="primary"
            href={job.url}
            target="_blank"
            rel="noopener noreferrer"
            onclick={onApplyClick}
          >
            Show <ArrowRight class="size-4" />
          </Button>
          {#if applied}
            <Badge variant="secondary"><Check class="mr-1 size-3.5" /> You applied</Badge>
          {/if}
        </div>
      </div>

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
      {#if job.remote && !e.work_mode}<Badge variant="secondary">Remote</Badge>{/if}
      <Badge variant="outline">{job.source}</Badge>
      {#if posted}<span class="text-xs text-muted-foreground">Posted {posted}</span>{/if}
    </div>

    {#if job.description}
      <!-- Description is server-sanitized HTML (see internal/sources), safe to render. -->
      <div class="job-description text-sm leading-relaxed">{@html job.description}</div>
    {/if}
  </article>
{/if}

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
