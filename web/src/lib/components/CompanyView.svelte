<script lang="ts">
  import { untrack } from 'svelte';
  import { api } from '$lib/api';
  import type { Company, Job } from '$lib/types';
  import { Paginator } from '$lib/paginated.svelte';
  import States from './States.svelte';
  import JobRow from './JobRow.svelte';
  import LoadMore from './LoadMore.svelte';
  import CompanyLogo from './CompanyLogo.svelte';

  // Company and its first page of jobs are server-rendered (route `load`), so the
  // header and rows are in the initial HTML. `slug` drives client-side paging.
  // The route remounts this component on slug change (#key), so the seed below
  // always reflects the current company.
  let {
    company,
    jobs,
    hasMore,
    slug,
  }: { company: Company; jobs: Job[]; hasMore: boolean; slug: string } = $props();

  const LIMIT = 20;

  // Seed the paginator from the server-rendered first page; "load more" fetches
  // subsequent pages client-side. The endpoint omits a total, so "more" is
  // inferred from a full page. The seed is an intentional one-time snapshot of
  // the initial props (untrack), not reactive state.
  const pager = new Paginator<Job>(async (limit, offset) => {
    const res = await api.getCompany(slug, limit, offset);
    return { items: res.jobs, hasMore: res.jobs.length === limit };
  }, LIMIT);
  pager.seed(untrack(() => ({ items: jobs, hasMore })));
</script>

<div class="flex items-center gap-3">
  <CompanyLogo name={company.name} size="size-8" />
  <h1 class="text-xl font-semibold tracking-tight">{company.name}</h1>
</div>

<div class="mt-4">
  {#if pager.items.length === 0}
    <States state="empty" message="No jobs for this company yet." />
  {:else}
    <div class="flex flex-col gap-3">
      {#each pager.items as job (job.public_slug)}
        <JobRow {job} />
      {/each}
    </div>

    {#if pager.hasMore}
      <LoadMore loading={pager.loadingMore} error={pager.loadMoreError} onclick={() => pager.loadMore()} />
    {/if}
  {/if}
</div>
