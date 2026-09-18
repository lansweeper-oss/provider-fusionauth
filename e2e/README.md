# E2E Tests

End-to-end tests run via [chainsaw](https://kyverno.github.io/chainsaw/) against
a local FusionAuth instance deployed in-cluster with PostgreSQL.

## Structure

```
e2e/
  setup/          # FusionAuth + PostgreSQL deployment manifests
  manifests/      # Shared resource manifests (apply-and-assert via uptest.mk in CI)
  tests/
    cluster/      # Tests for cluster-scoped resources (fusionauth.crossplane.io, application.fusionauth.crossplane.io)
    namespaced/   # Tests for namespaced resources (fusionauth.m.crossplane.io, application.fusionauth.m.crossplane.io)
```

## Running

```bash
make e2e  # runs chainsaw tests
```

## FusionAuth Setup

The e2e suite deploys:
- **PostgreSQL** (`postgres:16-bookworm`) as the database backend
- **FusionAuth** (`fusionauth/fusionauth-app:1.54.0`) with database search engine (no Elasticsearch)

An API key is bootstrapped automatically via FusionAuth's
[Kickstart](https://fusionauth.io/docs/get-started/download-and-install/development/kickstart)
feature on first boot. The static key `e2e-test-api-key-00000000000000000000`
is used by the provider credentials secret.

## Resources Tested

| Resource | Cluster Group | Namespaced Group |
|----------|---------------|------------------|
| Application | `fusionauth.crossplane.io/v1alpha1` | `fusionauth.m.crossplane.io/v1alpha1` |
| Role | `application.fusionauth.crossplane.io/v1alpha1` | `application.fusionauth.m.crossplane.io/v1alpha1` |

## Known Limitations

| Resource | Reason |
|----------|--------|
| OauthScope | Same pattern as Role (requires `applicationId`). Not yet tested — add once Role tests are validated at runtime. |
| Role / OauthScope `name` field | CRD may be missing the `name` field that the Terraform provider requires. Needs runtime validation. |
