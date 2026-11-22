# Railway pilot deployment

Deploy this directory as the root of the GitHub repository. The workflow assumes `main` is the default branch. It verifies pull requests into `main`, and deploys successful `main` pushes or manual runs selected on `main`.

## Railway service

1. In one Railway environment, provision PostgreSQL and an empty API service. Keep PostgreSQL on the private network. Use the API service as the GitHub Actions upload target. If the service already has a GitHub source, disable Railway autodeploys so GitHub Actions is the deploy trigger.
2. In the API service variables, set the values below. Use the exact HTTPS origin of the LIFF frontend, including any nondefault port and omitting the trailing slash. If the frontend is outside Railway's private network, give the API an HTTPS Railway domain for the server-side proxy.

   | Variable | Value |
   |---|---|
   | `DATABASE_URL` | Reference the database service, for example `${{Postgres.DATABASE_URL}}`. |
   | `WEB_APP_URL` | Exact LIFF web origin, for example `https://liff.example.com`. |
   | `LINE_LOGIN_CHANNEL_ID` | The real LINE Login channel ID. |
   | `JWT_SIGNING_KEY` | A random secret of at least 32 bytes. |
   | `PREVIEW_SIGNING_KEY` | A separate random secret of at least 32 bytes. |

   Generate each signing key separately with `openssl rand -hex 32`. Set these values in Railway. The API listens on Railway's injected `PORT`.
3. In **Settings → Deploy**, set the **Pre-Deploy Command** to `/app/railway-migrate` and the **Healthcheck Path** to `/health/ready`. The root Dockerfile is detected automatically. Leave the start command at the Dockerfile default. The migration command uses the private database and runs upward only. If it fails, Railway stops the rollout before sending traffic to the new instance.
4. Configure the LIFF frontend to proxy `/api/v1/*` to the backend from the frontend origin. Forward `Origin`, the request cookies, and `Authorization` when present, and preserve the API's `Set-Cookie` response. Run the planned LINE iOS/Android checks before collecting rider data.

[Railway PostgreSQL](https://docs.railway.com/databases/postgresql) · [Pre-deploy commands](https://docs.railway.com/deployments/pre-deploy-command) · [Railway healthchecks](https://docs.railway.com/deployments/healthchecks)

## GitHub Actions

Create a GitHub environment named `railway-pilot` and allow deployments from `main`. Add these settings under that environment.

| Type | Name | Value |
|---|---|---|
| Secret | `RAILWAY_TOKEN` | Railway project token scoped to the target environment |
| Variable | `RAILWAY_PROJECT_ID` | Railway project ID |
| Variable | `RAILWAY_ENVIRONMENT_ID` | Railway environment ID |
| Variable | `RAILWAY_SERVICE_ID` | API service ID |

The verification job runs static analysis, race-enabled tests, migration and readiness checks against temporary PostgreSQL, HTTP contract validation, and a container build and smoke check. The deploy job checks out the same revision, uploads it to Railway and waits for the attached deployment to finish. GitHub serializes deployments to this environment.

For a first rollout, check Railway's migration and deploy logs. If the API has an HTTPS domain, verify `GET /health/ready` returns 200 and that `/swagger/index.html` loads. On a failed migration, inspect the database version and failure before retrying. Rolling back an application deployment does not roll back the database schema.

[GitHub deployment environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments) · [Railway CLI deployment](https://docs.railway.com/cli/up)
