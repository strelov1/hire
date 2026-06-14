import { error } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { serverApi } from '$lib/server/api';
import type { PageServerLoad } from './$types';

const LIMIT = 20;

// Server-render the company and its first page of search results. The job list is
// search-backed and scoped to this company (company_slug), so it carries a true
// total (the vacancy count) and supports the same URL filters as /jobs. The
// company entity is fetched separately because search returns only jobs. A 404
// (unknown company) becomes a SvelteKit 404; other failures bubble to the 500 page.
export const load: PageServerLoad = async ({ params, url, fetch }) => {
  const client = serverApi(fetch);
  const facets = new URLSearchParams(url.searchParams);
  facets.set('company_slug', params.slug);
  try {
    const [{ company }, initial] = await Promise.all([
      client.getCompany(params.slug, 1, 0),
      client.searchJobs(facets, LIMIT, 0),
    ]);
    return { company, initial, slug: params.slug };
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) {
      error(404, 'Company not found');
    }
    throw e;
  }
};
