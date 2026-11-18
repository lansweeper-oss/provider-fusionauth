# Importing existing FusionAuth resources

You can bring objects that already exist in FusionAuth under Crossplane
without recreating them. You create a managed resource (MR) that points at the
existing object through the `crossplane.io/external-name` annotation, and the
provider reads the object instead of creating a new one.

The examples use the namespaced API (`fusionauth.m.crossplane.io`) with a
`ClusterProviderConfig` named `default`. The cluster-scoped API
(`fusionauth.crossplane.io`) works the same way: drop `metadata.namespace` and
use `providerConfigRef: {name: default}`.

## What goes in the external name

For every resource, the external name is the FusionAuth object ID, a UUID. Use
the bare ID, without the parent ID or any other prefix, even for child
resources such as `Role` and `OauthScope`.

These resources also accept a name instead of the ID:

| Kind | External name can be | Matched against |
|------|----------------------|-----------------|
| `User` | email or username | the login ID, within `forProvider.tenantId` when set |
| `Tenant` | tenant name | the exact name |
| `Application` | application name | the exact name, within `forProvider.tenantId` when set |
| `Group` | group name | the exact name, within `forProvider.tenantId` when set |
| `Role` | role name | the exact name, within `forProvider.applicationId` (required) |
| `OauthScope` | scope name | the exact name, within `forProvider.applicationId` (required) |
| `EntityTypePermission` | permission name | the exact name, within `forProvider.entityTypeId` (required) |
| `Connector` (`generic`, `ldap`) | connector name | the exact name, among connectors of the same type |
| `Messenger` (`generic`, `twilio`) | messenger name | the exact name, among messengers of the same type |
| `Lambda`, `Key`, `Theme`, `Email`, `Form`, `Consent`, `EntityType`, `Action` | object name | the exact name |

Any external name that is not a UUID is treated as a name for these kinds.
After the first successful read, the provider replaces the annotation
with the object ID, so later reconciles use the ID.

`Registration` is the exception to the ID rule. Its ID is built from
`forProvider.userId` and `forProvider.applicationId`, so set both fields and
use `<userId>:<applicationId>` as the external name.

## Finding the ID

When you need the ID, you can get it from the FusionAuth admin UI, where each
object's detail view shows its ID, or from the API:

```bash
# Tenants
curl -s -H "Authorization: $API_KEY" "$FUSIONAUTH_URL/api/tenant" | jq '.tenants[] | {id, name}'

# Applications (add X-FusionAuth-TenantId to scope to one tenant)
curl -s -H "Authorization: $API_KEY" "$FUSIONAUTH_URL/api/application" | jq '.applications[] | {id, name, tenantId}'

# A user by email or username
curl -s -H "Authorization: $API_KEY" "$FUSIONAUTH_URL/api/user?loginId=jdoe@example.com" | jq '.user.id'
```

Some IDs are the same on every FusionAuth install. The built-in `FusionAuth`
admin application is always `3c219e58-ed0e-4b18-ad48-f4f92793ae32`. The
`Default` tenant gets a random ID on each install, so look it up or import it
by name.

## Step 1: observe the object

Start with `managementPolicies: ["Observe"]`. In this mode the provider only
reads the object. It never creates, updates or deletes it, so a typo in the
external name cannot create a duplicate, and deleting the MR leaves the
FusionAuth object in place.

Import a user by email:

```yaml
apiVersion: fusionauth.m.crossplane.io/v1alpha1
kind: User
metadata:
  name: jdoe
  namespace: identity
  annotations:
    crossplane.io/external-name: jdoe@example.com
spec:
  managementPolicies: ["Observe"]
  forProvider: {}
  providerConfigRef:
    kind: ClusterProviderConfig
    name: default
```

Import the default tenant by name:

```yaml
apiVersion: fusionauth.m.crossplane.io/v1alpha1
kind: Tenant
metadata:
  name: default-tenant
  namespace: identity
  annotations:
    crossplane.io/external-name: Default
spec:
  managementPolicies: ["Observe"]
  forProvider: {}
  providerConfigRef:
    kind: ClusterProviderConfig
    name: default
```

Import a role by name. A child resource still needs its parent in
`forProvider`, and the role name is matched within that application. The
role ID works too:

```yaml
apiVersion: application.fusionauth.m.crossplane.io/v1alpha1
kind: Role
metadata:
  name: admin-role
  namespace: identity
  annotations:
    crossplane.io/external-name: admin
spec:
  managementPolicies: ["Observe"]
  forProvider:
    applicationId: 3c219e58-ed0e-4b18-ad48-f4f92793ae32
    name: admin
  providerConfigRef:
    kind: ClusterProviderConfig
    name: default
```

Check the result:

```bash
kubectl -n identity get user.fusionauth.m.crossplane.io jdoe \
  -o jsonpath='{.metadata.annotations.crossplane\.io/external-name}{"\n"}'
kubectl -n identity get user.fusionauth.m.crossplane.io jdoe -o yaml | yq '.status'
```

The MR is imported when it is `Ready=True` and `Synced=True`. The annotation
then holds the object ID, and `status.atProvider` shows the object as
FusionAuth returns it.

## Step 2: take over management

Observe-only is enough when you only need to reference the object or read its
fields. To let Crossplane change the object:

1. Change `managementPolicies`. The safest choice leaves out `Delete`, so
   that deleting the MR never deletes the FusionAuth object:

   ```yaml
   managementPolicies: ["Observe", "Create", "Update", "LateInitialize"]
   ```

   With `LateInitialize`, the provider fills empty `forProvider` fields with
   the values it reads from FusionAuth. Use `["*"]` only if you want
   `kubectl delete` on the MR to delete the object in FusionAuth too.
2. Review `spec.forProvider` after the next reconcile, then edit the fields
   you want to change.
3. Watch the MR. `Synced=True` with no further changes means the spec
   matches the object.

Do not import the `Default` tenant or the `FusionAuth` admin application with
`Delete` in the policies. FusionAuth needs both.

## How name lookups behave

- Names must match exactly, including case. `Defau` does not match `Default`.
- If more than one object matches, the read fails with an error that lists
  the matching IDs. For `Application` and `Group`, set
  `forProvider.tenantId` to narrow the match. Otherwise use the ID.
- Child kinds (`Role`, `OauthScope`, `EntityTypePermission`) need their
  parent ID in `forProvider`. Without it the read fails with
  `set applicationId to import by name` or `set entityTypeId to import by name`.
- If nothing matches, the provider reports that the object does not exist.
  With `["Observe"]` the MR stays `Synced=False` and nothing is created. With
  a policy that includes `Create`, Crossplane creates a new object, so check
  the name before you widen the policy.
- `Application` lookups only see active applications. Import an inactive
  application by ID.
- A user's email or username is unique only within a tenant. On an install
  with several tenants, set `forProvider.tenantId`. Otherwise FusionAuth may
  reject the lookup.

## Troubleshooting

### `Synced=False` with "external resource does not exist"

The ID or name matched nothing. Check the value against the API, and check
`forProvider.tenantId` if you set it.

### `cannot resolve "<name>" to an ID: name matches 2 objects`

Several objects share the name. For `Application` and `Group`, set
`forProvider.tenantId`. Otherwise use the ID.

### The annotation still holds the name after a while

The read has not succeeded yet, or the kind does not support names. Only the
kinds in the table under "What goes in the external name" resolve names.
Check the MR events with `kubectl describe`.

### Crossplane created a second object

The MR had `Create` in its policies and the external name did not match.
Delete the new object in FusionAuth, fix the external name, and start again
from step 1.
