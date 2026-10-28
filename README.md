# Provider FusionAuth

`provider-fusionauth` is a [Crossplane](https://crossplane.io/) provider that
manages [FusionAuth](https://fusionauth.io/) resources from Kubernetes. Built
with [Upjet](https://github.com/crossplane/upjet) code generation tools, it
wraps the upstream Terraform provider and exposes XRM-conformant managed
resources for applications, tenants, identity providers, keys, lambdas, themes,
and more.

[![Upstream Terraform Provider](https://img.shields.io/badge/upstream_terraform_provider-v1.3.8-blue?logo=terraform)](https://github.com/FusionAuth/terraform-provider-fusionauth/tree/v1.3.8)

## Getting Started

### Prerequisites

- A Kubernetes cluster with [Crossplane](https://crossplane.io/) installed
- A running FusionAuth instance with an API key

### Authentication

Create a Kubernetes secret containing your FusionAuth credentials as JSON:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: fusionauth-creds
  namespace: crossplane-system
type: Opaque
stringData:
  credentials: |
    {
      "host": "https://fusionauth.example.com",
      "api_key": "your-api-key"
    }
```

Then reference it in a `ProviderConfig`:

```yaml
apiVersion: fusionauth.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: fusionauth-creds
      namespace: crossplane-system
      key: credentials
```

## Developing

Run code-generation pipeline:

```console
go run cmd/generator/main.go .
```

Run against a Kubernetes cluster (out of cluster):

```console
make run
```

or (deploying in-cluster):

```console
make local-deploy
```

Run e2e tests (locally in a KinD cluster):

```console
# UPTEST_SKIP_DELETE=true
make e2e
```

Build, push, and install:

```console
make all
```

Build binary:

```console
make build
```

## Contributing

Refer to Crossplane's [CONTRIBUTING.md] file for more information on how the
Crossplane community prefers to work. The [Provider Development][provider-dev]
guide may also be of use.

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please
open an [issue].

## Code of conduct

This project follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

It also adheres to the [Crossplane AI Policy](https://github.com/crossplane/crossplane/blob/main/AI_POLICY.md).

## Licensing

`provider-fusionauth` is under the Apache 2.0 [license](LICENSE).

<!-- Links -->

[CONTRIBUTING.md]: https://github.com/crossplane/crossplane/blob/main/contributing
[issue]: https://github.com/lansweeper-oss/provider-fusionauth/issues
[provider-dev]: https://github.com/crossplane/crossplane/blob/master/contributing/guide-provider-development.md
