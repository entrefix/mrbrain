# Supabase Authentication Setup Guide

This document describes the environment variables and setup required for Supabase authentication integration.

## Backend Environment Variables

Create a `.env` file in the `backend/` directory with the following variables:

```env
# Supabase Configuration (Required)
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_JWT_SECRET=your-jwt-secret

# Database (Optional - defaults provided)
DATABASE_PATH=./data/todomyday.db

# Server (Optional - defaults provided)
PORT=8099
ALLOWED_ORIGINS=http://localhost:3111

# Other existing environment variables...
```

### How to Get Supabase Credentials

1. **SUPABASE_URL**: Found in your Supabase project settings under "API" → "Project URL"
2. **SUPABASE_ANON_KEY**: Found in your Supabase project settings under "API" → "Project API keys" → "anon public"
3. **SUPABASE_JWT_SECRET**: Found in your Supabase project settings under "Settings" → "API" → "JWT Secret"

### ⚠️ Security Warning: Service Role Key

**DO NOT** use the `SUPABASE_SERVICE_ROLE_KEY` in your application!

- The service role key grants **ADMIN access** and bypasses all Row Level Security (RLS) policies
- It should **NEVER** be stored in your `.env` file or used in the main application
- It should only be used in:
  - Secure backend migration scripts (run once, manually)
  - Admin-only CLI tools
  - Temporary troubleshooting (then immediately rotated)

The backend uses `SUPABASE_JWT_SECRET` to verify tokens, which is secure and correct. You do NOT need the service role key for normal authentication operations.

## Frontend Environment Variables

Create a `.env` file in the `frontend/` directory with the following variables:

```env
# Supabase Configuration (Required)
VITE_SUPABASE_URL=https://your-project.supabase.co
VITE_SUPABASE_ANON_KEY=your-anon-key

# API URL (Optional - defaults provided)
VITE_API_URL=http://localhost:8099
```

### How to Get Frontend Credentials

1. **VITE_SUPABASE_URL**: Same as backend SUPABASE_URL
2. **VITE_SUPABASE_ANON_KEY**: Same as backend SUPABASE_ANON_KEY

## Supabase Dashboard Configuration

### 1. Enable Email Provider

1. Go to Authentication → Providers
2. Enable "Email" provider
3. Configure email templates if needed

### 2. Enable Google OAuth Provider

1. Go to Authentication → Providers
2. Enable "Google" provider
3. Configure OAuth credentials:
   - Get Client ID and Client Secret from [Google Cloud Console](https://console.cloud.google.com/)
   - Create OAuth 2.0 credentials
   - Add authorized redirect URI: `https://your-project.supabase.co/auth/v1/callback`
   - Add authorized JavaScript origins: `https://your-project.supabase.co`

### 3. Configure Redirect URLs

1. Go to Authentication → URL Configuration
2. Add your site URL: `http://localhost:3111` (for development)
3. Add redirect URLs:
   - `http://localhost:3111/auth/callback` (for OAuth)
   - `http://localhost:3111/reset-password` (for password reset)

### 4. Email Templates (Optional)

1. Go to Authentication → Email Templates
2. Customize templates for:
   - Confirm signup
   - Reset password
   - Magic link

### 5. Email Confirmation (Development)

For development, you may want to disable email confirmation:
1. Go to Authentication → Settings
2. Toggle "Enable email confirmations" off

**Note**: In production, keep email confirmations enabled for security.

## User Migration

To migrate existing users to Supabase:

1. Ensure all environment variables are set
2. Run the migration script:

```bash
cd backend
go run scripts/migrate_users_to_supabase.go
```

**Important Notes:**
- The migration script creates users in Supabase with temporary passwords
- Users will need to use the "Forgot Password" flow to set their actual password
- The script links existing local users to Supabase users via `supabase_id`

## Testing

### Test Email/Password Authentication

1. Start the backend: `cd backend && go run cmd/server/main.go`
2. Start the frontend: `cd frontend && npm run dev`
3. Navigate to `http://localhost:3111/register`
4. Create an account with email/password
5. Sign in at `http://localhost:3111/login`

### Test Google OAuth

1. Ensure Google OAuth is configured in Supabase dashboard
2. Click "Continue with Google" on login/register pages
3. Complete OAuth flow
4. You should be redirected back to the app

### Test Password Reset

1. Go to `http://localhost:3111/forgot-password`
2. Enter your email
3. Check your email for the reset link
4. Click the link and set a new password

## Troubleshooting

### "Missing Supabase environment variables" error

- Ensure all required environment variables are set in `.env` files
- For frontend, variables must start with `VITE_` to be accessible

### OAuth redirect not working

- Check that redirect URLs are configured in Supabase dashboard
- Ensure the redirect URL matches exactly (including protocol and port)

### JWT verification fails

- Verify `SUPABASE_JWT_SECRET` matches the JWT secret in Supabase dashboard
- Check that tokens are being sent in the Authorization header as `Bearer <token>`

### User sync fails

- Check backend logs for errors
- Verify database connection
- Ensure `supabase_id` column exists in users table (run migrations)



