import { error } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { serverApi } from '$lib/server/api';
import type { PageServerLoad } from './$types';

const LIMIT = 20;

// Server-render the company and its first page of jobs. The endpoint omits a
// total, so "more" is inferred from a full page. A 404 becomes a SvelteKit 404
// page; other failures bubble to the 500 page.
export const load: PageServerLoad = async ({ params, fetch }) => {
  try {
    const { company, jobs } = await serverApi(fetch).getCompany(params.slug, LIMIT, 0);
    return { company, jobs, hasMore: jobs.length === LIMIT, slug: params.slug };
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) {
      error(404, 'Company not found');
    }
    throw e;
  }
};
