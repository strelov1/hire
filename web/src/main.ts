import { mount } from 'svelte';
import App from './App.svelte';
import './app.css';
import { initTheme } from '$lib/theme.svelte';
import { initAuth } from '$lib/auth.svelte';

// Apply the saved (or OS) theme once before mounting so there is no flash of
// the wrong palette.
initTheme();

// Validate any stored token in the background; the UI starts signed out and
// updates reactively once /me confirms (or discards) the token.
void initAuth();

const target = document.getElementById('app');
if (!target) throw new Error('#app element not found in index.html');

const app = mount(App, { target });

export default app;
