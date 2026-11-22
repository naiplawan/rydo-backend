# Changelog

## Unreleased

Initial pilot snapshot. The runtime OpenAPI document identifies this API as `0.2.0-local-pilot`.

### Added

- LINE sign-in, session refresh and logout, rider profiles, delivery platforms, and vehicle settings.
- Workday profit previews, confirmed and idempotent saves, edits, history, and daily reports.
- PostgreSQL migrations for the pilot data model, same-rider vehicle ownership, and finite financial values.
- Liveness and schema readiness endpoints, plus Swagger UI for the implemented API.
- A Railway container, pre-deploy database migrations, and GitHub Actions verification and deployment from `main`.

Before first use, configure the database, the LIFF frontend origin and LINE Login channel, and separate signing keys. See [deployment setup](DEPLOYMENT.md). The verification suite uses temporary PostgreSQL and a test double at the LINE verification boundary.
