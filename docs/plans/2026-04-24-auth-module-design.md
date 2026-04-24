# Auth Module & View Auth Enforcement — Design Document

**Date**: 2026-04-24
**Status**: Approved

---

## Overview

Create an embedded `auth` module for login/register/forgot/2FA pages, enforce authentication on all `/app/*` view routes by default, and make auth requirement configurable per module via `module.json`.

## Changes

### 1. ModuleDefinition — `auth` field
- Add `Auth *bool` to parser. Default `nil` = treated as `true` (require auth).
- `"auth": false` makes module views public.

### 2. Auth Module (embedded)
- `engine/embedded/modules/auth/` with login, register, forgot, reset, verify-2fa views/templates.
- `auth: false` in its own module.json (public).
- `menu_visibility: "none"` — never shows in sidebar.
- Settings: `register_enabled`, `otp_enabled`, `otp_channel`, `otp_type`.

### 3. Engine Auth Enforcement
- `handleViewGet`/`handleViewPost`: check module auth flag, redirect to `/app/auth/login?next=...` if unauthenticated.
- Sanitize `?next=` to only allow local `/app/...` paths.

### 4. menu_visibility: "none"
- Skip module entirely in menu builder.

### 5. Config
- `auth.register_enabled` in Viper config chain (TOML > YAML > env).
- Settings table overrides for runtime changes via admin UI.

### 6. Cleanup
- Remove `base/templates/views/login.html`.
- Remove hardcoded `loginPageHTML()` from app.go.
- `/app/login` redirects to `/app/auth/login` for backward compat.
