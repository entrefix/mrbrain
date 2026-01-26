import { getPostHog } from './posthog';

/**
 * Track an event in PostHog
 * @param eventName - Name of the event
 * @param properties - Optional properties to attach to the event
 */
export function trackEvent(eventName: string, properties?: Record<string, any>) {
  const posthog = getPostHog();
  if (posthog) {
    posthog.capture(eventName, properties);
    // Log in development for testing
    if (import.meta.env.DEV) {
      console.log('[Analytics] Event tracked:', eventName, properties);
    }
  } else {
    if (import.meta.env.DEV) {
      console.warn('[Analytics] PostHog not initialized, event not tracked:', eventName);
    }
  }
}

/**
 * Identify a user in PostHog
 * @param userId - Unique user identifier
 * @param traits - Optional user traits/properties
 */
export function identifyUser(userId: string, traits?: Record<string, any>) {
  const posthog = getPostHog();
  if (posthog) {
    posthog.identify(userId, traits);
    // Log in development for testing
    if (import.meta.env.DEV) {
      console.log('[Analytics] User identified:', userId, traits);
    }
  } else {
    if (import.meta.env.DEV) {
      console.warn('[Analytics] PostHog not initialized, user not identified:', userId);
    }
  }
}

/**
 * Reset user identification (on logout)
 */
export function resetUser() {
  const posthog = getPostHog();
  if (posthog) {
    posthog.reset();
  }
}
