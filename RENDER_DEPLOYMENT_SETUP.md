# Render Deployment with Conditional Supabase Migrations

## Overview
This GitHub Action automatically deploys your application to Render and only runs Supabase migrations if the Render deployment succeeds.

## Required GitHub Secrets

Add these secrets to your GitHub repository (Settings → Secrets and variables → Actions):

### Render Configuration
- `RENDER_API_KEY`: Your Render API key
  - Get from: [Render Dashboard → Account Settings → API Keys](https://dashboard.render.com/u/settings)
- `RENDER_SERVICE_ID`: The ID of your Render service to deploy
  - Get from: Your service URL in Render Dashboard (format: `srv-xxxxxxxxx`)

### Supabase Configuration  
- `SUPABASE_ACCESS_TOKEN`: Your Supabase access token
  - Get from: [Supabase Dashboard → Settings → Access Tokens](https://supabase.com/dashboard/account/tokens)
- `PRODUCTION_DB_PASSWORD`: Your production database password
- `PRODUCTION_PROJECT_ID`: Your Supabase project reference ID
  - Get from: Supabase Dashboard → Settings → General → Reference ID

## Workflow Behavior

### Trigger Conditions
- Automatically runs on push to `main` branch
- Can be manually triggered via GitHub Actions UI (`workflow_dispatch`)

### Deployment Process
1. **Deploy to Render**: Uses the `bounceapp/render-action` to deploy
   - Waits up to 10 minutes for deployment completion
   - Captures deployment URL for reference
2. **Apply Migrations**: Only runs if Render deployment succeeds
   - Links to production Supabase project
   - Applies all pending migrations with `supabase db push`
3. **Notifications**: Provides success/failure feedback with relevant details

### Safety Features
- **Conditional Execution**: Migrations only run after successful Render deployment
- **Timeout Protection**: 10-minute timeout prevents stuck deployments
- **Clear Feedback**: Success/failure messages with deploy URLs
- **Existing Migration Workflow**: Preserves your current migration deployment setup

## File Location
The workflow is defined in: `.github/workflows/deploy-render-with-migrations.yml`

## Testing
To test the workflow:
1. Add all required secrets to your GitHub repository
2. Push changes to the `main` branch or manually trigger the workflow
3. Monitor the Actions tab for deployment progress

## Troubleshooting
- **Render deployment timeout**: Increase `deploy-timeout` value (currently 600000ms = 10 minutes)
- **Migration failures**: Check Supabase project permissions and database connectivity
- **Missing secrets**: Verify all required secrets are properly configured in GitHub