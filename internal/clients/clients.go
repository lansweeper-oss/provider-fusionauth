package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	gofusionauth "github.com/FusionAuth/go-client/pkg/fusionauth"
	upstreamfusionauth "github.com/FusionAuth/terraform-provider-fusionauth/fusionauth"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/upjet/v2/pkg/terraform"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1beta1 "github.com/lansweeper-oss/provider-fusionauth/apis/cluster/v1beta1"
	namespacedv1beta1 "github.com/lansweeper-oss/provider-fusionauth/apis/namespaced/v1beta1"
)

const (
	errNoProviderConfig     = "no providerConfigRef provided"
	errGetProviderConfig    = "cannot get referenced ProviderConfig"
	errTrackUsage           = "cannot track ProviderConfig usage"
	errExtractCredentials   = "cannot extract credentials"
	errUnmarshalCredentials = "cannot unmarshal fusionauth credentials as JSON"
)

// TerraformSetupBuilder returns a terraform.SetupFn that configures the
// Terraform provider with credentials extracted from the ProviderConfig.
func TerraformSetupBuilder() terraform.SetupFn {
	return func(ctx context.Context, crClient client.Client, mg resource.Managed) (terraform.Setup, error) {
		ps := terraform.Setup{}

		pcSpec, err := resolveProviderConfig(ctx, crClient, mg)
		if err != nil {
			return terraform.Setup{}, fmt.Errorf("cannot resolve provider config: %w", err)
		}

		data, err := resource.CommonCredentialExtractor(ctx, pcSpec.Credentials.Source, crClient, pcSpec.Credentials.CommonCredentialSelectors)
		if err != nil {
			return terraform.Setup{}, fmt.Errorf(errExtractCredentials+": %w", err)
		}
		creds := map[string]any{}
		if err := json.Unmarshal(data, &creds); err != nil {
			return terraform.Setup{}, fmt.Errorf(errUnmarshalCredentials+": %w", err)
		}

		ps.Configuration = creds

		host, _ := creds["host"].(string)
		apiKey, _ := creds["api_key"].(string)
		if host == "" || apiKey == "" {
			return terraform.Setup{}, fmt.Errorf("host and api_key are required in credentials")
		}

		hostURL, err := url.Parse(host)
		if err != nil {
			return terraform.Setup{}, fmt.Errorf("cannot parse host URL: %w", err)
		}

		faClient := gofusionauth.NewClientWithRetryConfiguration(
			&http.Client{Timeout: 30 * time.Second},
			hostURL,
			apiKey,
			gofusionauth.RetryConfigurationFromEnv(),
		)

		ps.Meta = upstreamfusionauth.Client{
			FAClient: *faClient,
			Host:     host,
			APIKey:   apiKey,
		}

		return ps, nil
	}
}

func toSharedPCSpec(pc *clusterv1beta1.ProviderConfig) (*namespacedv1beta1.ProviderConfigSpec, error) {
	if pc == nil {
		return nil, nil
	}
	data, err := json.Marshal(pc.Spec)
	if err != nil {
		return nil, err
	}

	var mSpec namespacedv1beta1.ProviderConfigSpec
	err = json.Unmarshal(data, &mSpec)
	return &mSpec, err
}

func resolveProviderConfig(ctx context.Context, crClient client.Client, mg resource.Managed) (*namespacedv1beta1.ProviderConfigSpec, error) {
	switch managed := mg.(type) {
	case resource.LegacyManaged: //nolint:staticcheck
		return resolveLegacy(ctx, crClient, managed)
	case resource.ModernManaged:
		return resolveModern(ctx, crClient, managed)
	default:
		return nil, errors.New("resource is not a managed resource")
	}
}

func resolveLegacy(ctx context.Context, client client.Client, mg resource.LegacyManaged) (*namespacedv1beta1.ProviderConfigSpec, error) { //nolint:staticcheck
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}
	pc := &clusterv1beta1.ProviderConfig{}
	if err := client.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
		return nil, fmt.Errorf(errGetProviderConfig+": %w", err)
	}

	t := resource.NewLegacyProviderConfigUsageTracker(client, &clusterv1beta1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, fmt.Errorf(errTrackUsage+": %w", err)
	}

	return toSharedPCSpec(pc)
}

func resolveModern(ctx context.Context, crClient client.Client, mg resource.ModernManaged) (*namespacedv1beta1.ProviderConfigSpec, error) {
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}

	pcRuntimeObj, err := crClient.Scheme().New(namespacedv1beta1.SchemeGroupVersion.WithKind(configRef.Kind))
	if err != nil {
		return nil, fmt.Errorf("unknown GVK for ProviderConfig"+": %w", err)
	}
	pcObj, ok := pcRuntimeObj.(client.Object)
	if !ok {
		return nil, fmt.Errorf("provider config type %T is not a client.Object; this indicates a code generation issue", pcRuntimeObj)
	}

	if err := crClient.Get(ctx, types.NamespacedName{Name: configRef.Name, Namespace: mg.GetNamespace()}, pcObj); err != nil {
		return nil, fmt.Errorf(errGetProviderConfig+": %w", err)
	}

	var pcSpec namespacedv1beta1.ProviderConfigSpec
	pcu := &namespacedv1beta1.ProviderConfigUsage{}
	switch pc := pcObj.(type) {
	case *namespacedv1beta1.ProviderConfig:
		pcSpec = pc.Spec
		if pcSpec.Credentials.SecretRef != nil {
			pcSpec.Credentials.SecretRef.Namespace = mg.GetNamespace()
		}
	case *namespacedv1beta1.ClusterProviderConfig:
		pcSpec = pc.Spec
	default:
		return nil, errors.New("unknown provider config type")
	}
	t := resource.NewProviderConfigUsageTracker(crClient, pcu)
	if err := t.Track(ctx, mg); err != nil {
		return nil, fmt.Errorf(errTrackUsage+": %w", err)
	}
	return &pcSpec, nil
}
