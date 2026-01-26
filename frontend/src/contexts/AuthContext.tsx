import { createContext, useContext, useEffect, useState } from 'react';
import { supabase } from '../lib/supabase';
import { authApi } from '../api';
import { User } from '../types';
import type { AuthError, Session } from '@supabase/supabase-js';
import { identifyUser, resetUser, trackEvent } from '../utils/analytics';

interface AuthContextType {
  user: User | null;
  loading: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signInWithGoogle: () => Promise<void>;
  signOut: () => Promise<void>;
  resetPassword: (email: string) => Promise<void>;
  updatePassword: (newPassword: string) => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // Sync user profile from backend (only called after login or when explicitly needed)
  const syncUserFromBackend = async () => {
    try {
      const backendUser = await authApi.me();
      setUser(backendUser);
      // Identify user in PostHog
      if (backendUser) {
        identifyUser(backendUser.id, {
          email: backendUser.email,
          full_name: backendUser.full_name,
        });
      }
    } catch (error: any) {
      console.error('Failed to sync user from backend:', error);
      // If we get a 401, the session is invalid - clear user
      if (error?.response?.status === 401) {
        setUser(null);
        resetUser(); // Reset PostHog user identification
      }
      // For other errors, don't clear user (might be network issue)
      // Don't throw - handled internally
    }
  };

  useEffect(() => {
    let mounted = true;

    // Check for existing session FIRST
    supabase.auth.getSession().then(({ data: { session } }) => {
      if (!mounted) return;

      if (session) {
        syncUserFromBackend().finally(() => {
          if (mounted) setLoading(false);
        });
      } else {
        setUser(null);
        setLoading(false);
      }
    });

    // Listen for auth state changes (NO ASYNC - prevents Supabase deadlock)
    const { data: { subscription } } = supabase.auth.onAuthStateChange(
      (event, session) => {
        if (!mounted) return;

        if (event === 'SIGNED_IN' && session) {
          // Trigger sync without awaiting in callback
          syncUserFromBackend();
        } else if (event === 'SIGNED_OUT') {
          setUser(null);
        } else if (event === 'TOKEN_REFRESHED' && session) {
          // Also sync on token refresh to ensure user data is fresh
          syncUserFromBackend();
        }
      }
    );

    return () => {
      mounted = false;
      subscription.unsubscribe();
    };
  }, []);

  const signIn = async (email: string, password: string) => {
    const { data, error } = await supabase.auth.signInWithPassword({
      email,
      password,
    });

    if (error) {
      trackEvent('login_failed', { error: error.message });
      throw error;
    }

    // Session is set by Supabase, onAuthStateChange will trigger syncUserFromBackend
    // But we can also sync immediately for better UX
    if (data.session) {
      await syncUserFromBackend();
      trackEvent('user_logged_in', { method: 'email' });
    }
  };

  const signUp = async (email: string, password: string) => {
    const { data, error } = await supabase.auth.signUp({
      email,
      password,
    });

    if (error) {
      trackEvent('registration_failed', { error: error.message });
      throw error;
    }

    // If email confirmation is disabled, user is immediately signed in
    if (data.session) {
      await syncUserFromBackend();
      trackEvent('user_registered', { method: 'email' });
    } else {
      trackEvent('user_registered', { method: 'email', requires_confirmation: true });
    }
  };

  const signInWithGoogle = async () => {
    trackEvent('login_initiated', { method: 'google' });
    const { error } = await supabase.auth.signInWithOAuth({
      provider: 'google',
      options: {
        redirectTo: `${window.location.origin}/auth/callback`,
      },
    });

    if (error) {
      trackEvent('login_failed', { method: 'google', error: error.message });
      throw error;
    }
  };

  const signOut = async () => {
    const { error } = await supabase.auth.signOut();
    if (error) {
      throw error;
    }
    setUser(null);
    resetUser(); // Reset PostHog user identification
    trackEvent('user_logged_out');
  };

  const resetPassword = async (email: string) => {
    const { error } = await supabase.auth.resetPasswordForEmail(email, {
      redirectTo: `${window.location.origin}/reset-password`,
    });

    if (error) {
      throw error;
    }
  };

  const updatePassword = async (newPassword: string) => {
    const { error } = await supabase.auth.updateUser({
      password: newPassword,
    });

    if (error) {
      throw error;
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        signIn,
        signUp,
        signInWithGoogle,
        signOut,
        resetPassword,
        updatePassword,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
