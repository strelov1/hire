<script lang="ts">
  import { onMount } from 'svelte';
  import { listCompanies } from '$lib/api';
  import { Paginator } from '$lib/paginated.svelte';
  import { Badge } from '$lib/ui';
  import States from './States.svelte';
  import LoadMore from './LoadMore.svelte';
  import CompanyLogo from './CompanyLogo.svelte';

  const companies = new Paginator(listCompanies);
  onMount(() => companies.start());
</script>

{#if companies.status === 'loading'}
  <States state="loading" />
{:else if companies.status === 'error'}
  <States state="error" message="Failed to load companies." />
{:else if companies.items.length === 0}
  <States state="empty" message="No companies yet." />
{:else}
  <div class="flex flex-col gap-3">
    {#each companies.items as company (company.slug)}
      <a
        href={`/companies/${company.slug}`}
        class="flex items-center justify-between rounded-lg border border-border px-4 py-3 transition-colors hover:bg-accent"
      >
        <span class="flex min-w-0 items-center gap-2.5">
          <CompanyLogo name={company.name} size="size-6" />
          <span class="truncate font-medium">{company.name}</span>
        </span>
        <Badge variant="outline">{company.job_count} jobs</Badge>
      </a>
    {/each}
  </div>

  {#if companies.hasMore}
    <LoadMore
      loading={companies.loadingMore}
      error={companies.loadMoreError}
      onclick={() => companies.loadMore()}
    />
  {/if}
{/if}
