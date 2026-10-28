package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Terraform read functions of these resources look up the object by the
// bare object ID. If the Terraform ID has a parent prefix, the resource is
// not found after an import or after a provider restart.
func TestExternalNameIsBareTerraformID(t *testing.T) {
	params := map[string]any{
		"application_id": "app-id",
		"entity_type_id": "entity-type-id",
		"tenant_id":      "tenant-id",
	}
	for _, r := range []string{
		"fusionauth_application_oauth_scope",
		"fusionauth_application_role",
		"fusionauth_entity",
		"fusionauth_entity_type_permission",
	} {
		t.Run(r, func(t *testing.T) {
			e := ExternalNameConfigs[r]
			id, err := e.GetIDFn(context.Background(), "object-id", params, nil)
			require.NoError(t, err)
			assert.Equal(t, "object-id", id)
			name, err := e.GetExternalNameFn(map[string]any{"id": "object-id"})
			require.NoError(t, err)
			assert.Equal(t, "object-id", name)
		})
	}
}
