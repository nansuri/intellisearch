# PRD — Security & Business Rules

> Part of the [PRD index](index.md).

- **Roles & access:** only the Super Owner can access the Owner Control Panel; General Users only use the main page and their own account settings.
- **AI usage limits:** each user has a daily question quota and a per-user rate limit; reaching the limit shows a friendly message.
- **URL submission rate limit:** rate limit the URL submission feature to prevent abuse of the scrapers.
- **Secrets vaulting:** ensure API keys for Telegram are securely vaulted.
- **QR payment codes:** QR payment codes should be time-limited and signed.
- **Branding:** the site name, logo, and tagline shown on the public main page are configured by the Super Owner via the Owner Control Panel; the codebase ships with a generic default and contains no hardcoded brand.

> Implementation of these rules lives in the [SDD — Security](../SDD/security.md).