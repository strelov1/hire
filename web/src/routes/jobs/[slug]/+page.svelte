<script lang="ts">
  import JobView from '$lib/components/JobView.svelte';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();
</script>

<svelte:head>
  <!-- Minimal job-specific title; full metadata/canonical/OG + JSON-LD land in
       the SEO tasks (5.1/5.2). Replaces the static "freehire" title. -->
  <title>{data.job.title} — {data.job.company || 'freehire'}</title>
</svelte:head>

<div class="mx-auto w-full max-w-3xl px-4 py-6">
  <JobView job={data.job} />
</div>
