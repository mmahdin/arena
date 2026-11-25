# Theia Auth and Kavenegar — local and staging

Theia uses its own Keycloak 26.8.2 image, `theia-users` / `theia-staff` realms,
and PostgreSQL database. The existing shared `ramona-platform-auth` service,
its database and its realms are not changed. This setup enables authenticated
internal testing; it does not change the product's data-use scope.

## Local restart

The private backend `.env.auth` supplies:

```dotenv
THEIA_SMS_ENV=local
THEIA_SMS_TRANSPORT=kavenegar
THEIA_SMS_KAVENEGAR_API_KEY=<private provider key>
THEIA_SMS_KAVENEGAR_TEMPLATE=loginotp
```

`loginotp` is the existing Ramona authenticator's default verify/lookup template.
Confirm that template is enabled for this provider account before the first
delivery test; change the private environment if the account uses another name.
Never put a key in a realm JSON, Docker build argument, frontend variable or Git.
The frontend's only local auth setting is the public, nonsecret
`VITE_KEYCLOAK_ORIGIN=http://auth.localhost:8180` in its ignored `.env.local`.

The local launcher passes both `.env` and `.env.auth` to Compose and recreates
Keycloak when its environment changes. Existing PostgreSQL data, clients and
users are retained. Run `./theia-dev.sh` from the backend root, or the existing
workspace launcher, and open **http://app.localhost:8080**. Phone challenges now
use the provider; the old mock outbox is not the source of new codes.

## Staging files and private prerequisites

The backend image `infra/keycloak/Dockerfile.Staging` builds the SPI and theme,
prepares optimized Keycloak and imports only the staging realm templates.
The User realm starts with exactly one required SMS execution, confidential
PKCE S256 client and the signed `theia_sms_verified` session-note mapper.
Staff login remains disabled in the backend configuration until separately used.

Public origins:

- Auth: `https://ai-therapist-auth.alibaba.stg.agidp.ir`
- User: `https://ai-therapist-frontend.alibaba.stg.agidp.ir`
- Staff: `https://theia-backoffice.alibaba.stg.agidp.ir` (reserved; login disabled)

`launchpad-ramona/deployments/ai-therapist-auth.yml` publishes only the two
realm prefixes and static resources. The management port and Admin API are
not routed publicly. Its health probes use the internal management port 9000.
`manifests/ai-therapist-auth-postgres.yaml` creates a dedicated persistent
PostgreSQL StatefulSet, a headless private Service and an ingress NetworkPolicy
allowing only the Auth pods. A default StorageClass must be available for its PVC.

Prepare credentials once on the operator machine, from the backend root:

```sh
python3 scripts/auth/prepare_staging_auth.py
```

All generated files are under ignored `.local/staging-auth` with directory mode
0700 and file mode 0600. Separate staging database/admin credentials, BFF
transaction key, SMS HMAC key and two client secrets are generated once and
reused on subsequent runs. The provider key comes from private `.env.auth`.
Keep these generated credentials in the operator's approved secret store;
losing them is not a reason to rotate an existing database or realm implicitly.

With an explicitly configured **staging** Kubernetes context, apply prerequisites:

```sh
python3 scripts/auth/prepare_staging_auth.py --apply \
  --context YOUR_STAGING_CONTEXT \
  --launchpad-root /path/to/launchpad-ramona
```

The helper requires `kubectl` access to the named namespace resources. Keycloak
uses its full canonical HTTPS hostname without forwarded-header parsing, so no
Gateway/Node/CNI lookup is needed. For the Go BFF it reads only currently ready
frontend Pod IPs and trusts those exact peers. If frontend Pods are replaced,
rerun this step and reload the backend before accepting their BFF requests.
An operator may instead supply independently verified private proxy ranges with
`--trusted-proxy-cidrs`; do not invent a cluster range or trust `0.0.0.0/0`.
A prepare-only run uses a loopback placeholder and needs an apply step.

A namespace credential restricted to ConfigMap/Secret management can apply only
those resources, without trying to create or restart workloads:

```sh
KUBECONFIG=/path/to/alibaba-stage.yaml \
python3 scripts/auth/prepare_staging_auth.py --apply --config-only \
  --context alibaba-stage --provider-env .local/staging-auth/cluster-provider.env
```

For the same Kavenegar account as Ramona, populate that private provider file from
`ramona-platform-auth-secret` key `KAVENEGAR_API_KEY`, mapped to
`THEIA_SMS_KAVENEGAR_API_KEY`. Read/copy it privately; never print or commit it.
This mode still preserves existing credentials and merge-patches only backend
Auth settings. PostgreSQL/Service/NetworkPolicy and Auth deployment remain the
deployment operator's or CI's responsibility when the namespace role forbids
creating workloads.

The apply step installs the new Auth/PostgreSQL secrets and configmaps, waits
for the dedicated database, and **merge-patches only auth fields** in the existing
backend configmap/secret. Existing AI, SQL Server, provider and encryption
configuration is preserved. Frontend config supplies its private API upstream
and `PUBLIC_SCHEME=https`; its staging build embeds only the public auth origin.

Then run the dedicated **`deploy staging-v2-ai-therapist-auth`** job in the backend
pipeline. The Auth image builds from the backend source repository under the
separate `ai-therapist-auth` image/release name. After Auth discovery is ready,
apply backend migrations 0041/0042 as required by the AI release and run the
backend Deploy job. The independent frontend pipeline deploys its Nginx image.
This Auth setup does not migrate, sign or replace the AI runtime/bundle: the
existing AI deployment still needs its own valid v4 configuration and bundle.

Keycloak startup import skips realms already present. Changes to an existing
staging realm need a reviewed Admin update; rebuilding the image is not a realm
migration. Do not delete an existing realm or PostgreSQL volume to apply config.

## Verification

```sh
docker build -f infra/keycloak/Dockerfile.Staging -t theia-keycloak-staging:auth-kavenegar .
python3 scripts/auth/smoke_staging_image.py
python3 -m unittest discover -s scripts/tests -p 'test_*auth*.py'
python3 scripts/auth/activate_user_bff.py  # read-only local realm recheck
```

The image smoke uses a disposable PostgreSQL, fake provider key and random
test credentials; it verifies actual realm import, exact HTTPS issuers,
callback, SMS-only execution, proof mapper, optional email/name and real phone-form
GET/invalid POST without proxy-header trust. It sends
no SMS and does not establish provider delivery. A successful `account/info`
provider check establishes key access, not template availability or delivery.
Finish delivery testing manually with the intended test recipient in the app.

References: [Keycloak container build](https://www.keycloak.org/server/containers),
[realm import](https://www.keycloak.org/server/importExport),
[Kavenegar API](https://kavenegar.github.io/kavenegar_en/).
