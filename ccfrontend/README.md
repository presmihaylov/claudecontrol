# ccfrontend

Modern Next.js 15 frontend application for Claude Control, providing a web-based dashboard for managing AI agent integrations and organization settings.

## Overview

The ccfrontend is a full-stack Next.js application featuring:
- **Modern Tech Stack**: Next.js 15 with React 19 and TypeScript
- **Authentication**: Clerk integration with protected routes
- **Styling**: Tailwind CSS 4 with Shadcn/ui components
- **Development Tools**: Biome for fast linting and formatting
- **HTTPS Development**: Secure local development environment

## Prerequisites

- Node.js 18+ or Bun (recommended)
- ccbackend running on localhost:8080

## Installation

```bash
# Install dependencies (recommended: use Bun)
bun install
```

## Environment Configuration

Create a `.env.local` file in the ccfrontend directory:

```env
# Clerk Authentication
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_your_clerk_publishable_key
CLERK_SECRET_KEY=sk_test_your_clerk_secret_key

# Backend API Configuration
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080

# Optional: Custom domain configuration
NEXT_PUBLIC_CLERK_SIGN_IN_URL=/sign-in
NEXT_PUBLIC_CLERK_SIGN_UP_URL=/sign-up
NEXT_PUBLIC_CLERK_AFTER_SIGN_IN_URL=/dashboard
NEXT_PUBLIC_CLERK_AFTER_SIGN_UP_URL=/dashboard
```

### Required Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` | Clerk publishable key for client-side auth | Yes |
| `CLERK_SECRET_KEY` | Clerk secret key for server-side operations | Yes |
| `NEXT_PUBLIC_API_BASE_URL` | Backend API base URL | Yes |

## Development Commands

### Development Server

```bash
# Run development server with HTTPS (recommended)
bun dev

# Run development server with HTTP
bun run dev:http

# Alternative with npm
npm run dev
```

The development server will start on:
- **HTTPS**: https://localhost:3000 (default)
- **HTTP**: http://localhost:3000 (with dev:http)

### Build and Production

```bash
# Build production bundle
bun run build

# Start production server
bun start
```

