<script lang="ts">
  import CompanyView from '$lib/components/CompanyView.svelte';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();
</script>

<svelte:head>
  <!-- Minimal company-specific title; full metadata + Organization JSON-LD land
       in the SEO tasks (5.1/5.2). -->
  <title>{data.company.name} — freehire</title>
</svelte:head>

<div class="mx-auto w-full max-w-3xl px-4 py-6">
  <!-- Remount on slug change so the seeded paginator starts fresh per company. -->
  {#key data.slug}
    <CompanyView company={data.company} jobs={data.jobs} hasMore={data.hasMore} slug={data.slug} />
  {/key}
</div>
