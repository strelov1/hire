<script lang="ts">
  import { onMount } from 'svelte';
  import { listJobs, searchJobs } from '$lib/api';
  import { Paginator } from '$lib/paginated.svelte';
  import States from './States.svelte';
  import JobRow from './JobRow.svelte';
  import LoadMore from './LoadMore.svelte';

  let query = $state('');
  // The paginator is rebuilt when the active query changes: a non-empty query
  // searches, an empty one falls back to the full list. Search results share the
  // Job wire shape, so the same JobRow renders both.
  let jobs = $state(new Paginator(listJobs));

  function run(raw: string) {
    const q = raw.trim();
    jobs = q ? new Paginator((limit, offset) => searchJobs(q, limit, offset)) : new Paginator(listJobs);
    jobs.start();
  }

  // Debounce input so we issue one request after typing settles, not per keystroke.
  let timer: ReturnType<typeof setTimeout>;
  function onInput() {
    clearTimeout(timer);
    timer = setTimeout(() => run(query), 300);
  }

  onMount(() => {
    jobs.start();
    // Cleanup: a debounce timer left running after unmount would fire run()
    // and start a fetch for a component that no longer exists.
    return () => clearTimeout(timer);
  });
</script>

<div class="mb-4">
  <input
    type="search"
    bind:value={query}
    oninput={onInput}
    placeholder="Search jobs…"
    aria-label="Search jobs"
    class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
  />
</div>

{#if jobs.status === 'loading'}
  <States state="loading" />
{:else if jobs.status === 'error'}
  <States state="error" message="Failed to load jobs." />
{:else if jobs.items.length === 0}
  <States state="empty" message={query.trim() ? 'No matching jobs.' : 'No jobs yet.'} />
{:else}
  <div class="flex flex-col gap-3">
    {#each jobs.items as job (job.public_slug)}
      <JobRow {job} />
    {/each}
  </div>

  {#if jobs.hasMore}
    <LoadMore loading={jobs.loadingMore} error={jobs.loadMoreError} onclick={() => jobs.loadMore()} />
  {/if}
{/if}
