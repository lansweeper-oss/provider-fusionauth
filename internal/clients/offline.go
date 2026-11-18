package clients

import (
	"context"

	upstreamfusionauth "github.com/FusionAuth/terraform-provider-fusionauth/fusionauth"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/upjet/v2/pkg/terraform"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OfflineTerraformSetupBuilder returns the terraform.SetupFn used by the diff
// server. It ignores the ProviderConfig entirely: no credentials are read and
// no FusionAuth client is built.
//
// The diff server plans from local state, so the meta only reaches
// CustomizeDiff, and the only one upstream (fusionauth_api_key) does not use
// it. An unconfigured client is therefore enough.
func OfflineTerraformSetupBuilder() terraform.SetupFn {
	return func(context.Context, client.Client, resource.Managed) (terraform.Setup, error) {
		return terraform.Setup{
			Configuration: map[string]any{},
			Meta:          upstreamfusionauth.Client{},
		}, nil
	}
}
