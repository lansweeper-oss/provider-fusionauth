package config

import (
	"context"
	"fmt"
	"net/http"

	gofusionauth "github.com/FusionAuth/go-client/pkg/fusionauth"
	upstreamfusionauth "github.com/FusionAuth/terraform-provider-fusionauth/fusionauth"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// lookupFn resolves a human-readable key (a name, an email) to the object ID.
// It returns "" when no object matches.
type lookupFn func(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, key string) (string, error)

// readWithLookup wraps an upstream read so that the external name can be a
// human-readable key instead of the object ID. A key that is not a UUID is
// resolved with lookup before the upstream read runs. Upjet then writes the
// object ID back to the external-name annotation.
func readWithLookup(upstream schema.ReadContextFunc, lookup lookupFn) schema.ReadContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
		key := d.Id()
		if key == "" || uuid.Validate(key) == nil {
			return upstream(ctx, d, m)
		}

		c, ok := m.(upstreamfusionauth.Client)
		if !ok {
			return diag.Errorf("unexpected provider meta type %T", m)
		}
		id, err := lookup(ctx, d, c, key)
		if err != nil {
			return diag.FromErr(fmt.Errorf("cannot resolve %q to an ID: %w", key, err))
		}
		d.SetId(id)
		if id == "" {
			return nil
		}
		return upstream(ctx, d, m)
	}
}

// lookupUserByLoginID resolves an email or a username to the user ID.
func lookupUserByLoginID(_ context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, loginID string) (string, error) {
	// A login ID is only unique within a tenant.
	if tenantID, ok := d.Get("tenant_id").(string); ok && tenantID != "" {
		c.FAClient.TenantId = tenantID
	}
	resp, faErrs, err := c.FAClient.RetrieveUserByLoginId(loginID)
	if err != nil {
		return "", err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return resp.User.Id, nil
	case http.StatusNotFound:
		return "", nil
	default:
		return "", fmt.Errorf("status %d: %v", resp.StatusCode, faErrs)
	}
}

// lookupTenantByName resolves a tenant name to the tenant ID.
func lookupTenantByName(_ context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, err := c.FAClient.RetrieveTenants()
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return uniqueID(resp.Tenants, func(t gofusionauth.Tenant) (string, bool) {
		return t.Id, t.Name == name
	})
}

// lookupApplicationByName resolves an application name to the application
// ID. A name is matched within spec.forProvider.tenantId when it is set.
func lookupApplicationByName(_ context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	tenantID, _ := d.Get("tenant_id").(string)
	resp, err := c.FAClient.RetrieveApplications()
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return uniqueID(resp.Applications, func(a gofusionauth.Application) (string, bool) {
		return a.Id, a.Name == name && (tenantID == "" || a.TenantId == tenantID)
	})
}

// lookupGroupByName resolves a group name to the group ID. A name is matched
// within spec.forProvider.tenantId when it is set.
func lookupGroupByName(_ context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	tenantID, _ := d.Get("tenant_id").(string)
	resp, err := c.FAClient.RetrieveGroups()
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return uniqueID(resp.Groups, func(g gofusionauth.Group) (string, bool) {
		return g.Id, g.Name == name && (tenantID == "" || g.TenantId == tenantID)
	})
}

// uniqueID returns the ID of the only item that matches. It returns "" when
// no item matches, and an error when more than one matches, so that an
// ambiguous name never resolves to an arbitrary object.
func uniqueID[T any](items []T, match func(T) (string, bool)) (string, error) {
	var found []string
	for _, item := range items {
		if id, ok := match(item); ok {
			found = append(found, id)
		}
	}
	if len(found) > 1 {
		return "", fmt.Errorf("name matches %d objects %v: use the ID or set tenantId", len(found), found)
	}
	if len(found) == 0 {
		return "", nil
	}
	return found[0], nil
}
