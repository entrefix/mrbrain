import posthog from 'posthog-js';

let isInitialized = false;

export function initPostHog() {
  // Only initialize if not already initialized and keys are present
  if (isInitialized) {
    return;
  }

  const apiKey = import.meta.env.VITE_POSTHOG_KEY;
  const host = import.meta.env.VITE_POSTHOG_HOST || 'https://app.posthog.com';

  // Only initialize if API key is provided
  if (!apiKey) {
    console.log('[PostHog] API key not found, skipping initialization');
    return;
  }

  try {
    posthog.init(apiKey, {
      api_host: host,
      loaded: (posthog) => {
        console.log('[PostHog] ✅ Initialized successfully');
        console.log('[PostHog] API Host:', host);
        console.log('[PostHog] Ready to track events');
      },
      capture_pageview: true,
      capture_pageleave: true,
      // Enable debug mode in development
      debug: import.meta.env.DEV,
    });
    isInitialized = true;
    console.log('[PostHog] Initialization started...');
  } catch (error) {
    console.error('[PostHog] ❌ Failed to initialize:', error);
  }
}

export function getPostHog() {
  return isInitialized ? posthog : null;
}
