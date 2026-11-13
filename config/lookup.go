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
// human-readable key instead of the object ID.
// A key that is not a UUID is resolved with lookup before the upstream read runs.
// Upjet then writes the object ID back to the external-name annotation.
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
func lookupUserByLoginID(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, loginID string) (string, error) {
	// A login ID is only unique within a tenant.
	if tenantID, ok := d.Get("tenant_id").(string); ok && tenantID != "" {
		c.FAClient.TenantId = tenantID
	}
	resp, faErrs, err := c.FAClient.RetrieveUserByLoginIdWithContext(ctx, loginID)
	if err != nil {
		return "", err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return resp.User.Id, nil
	case http.StatusNotFound:
		return "", nil
	}
	if faErrs != nil {
		return "", fmt.Errorf("status %d: %w", resp.StatusCode, *faErrs)
	}
	return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
}

// lookupTenantByName resolves a tenant name to the tenant ID.
func lookupTenantByName(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, err := c.FAClient.RetrieveTenantsWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.Tenants, func(t gofusionauth.Tenant) (string, bool) {
		return t.Id, t.Name == name
	})
}

// lookupApplicationByName resolves an application name to the application ID.
// A name is matched within spec.forProvider.tenantId when it is set.
func lookupApplicationByName(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	tenantID, _ := d.Get("tenant_id").(string)
	resp, err := c.FAClient.RetrieveApplicationsWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.Applications, func(a gofusionauth.Application) (string, bool) {
		return a.Id, a.Name == name && (tenantID == "" || a.TenantId == tenantID)
	})
}

// lookupGroupByName resolves a group name to the group ID.
// A name is matched within spec.forProvider.tenantId when it is set.
func lookupGroupByName(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	tenantID, _ := d.Get("tenant_id").(string)
	resp, err := c.FAClient.RetrieveGroupsWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.Groups, func(g gofusionauth.Group) (string, bool) {
		return g.Id, g.Name == name && (tenantID == "" || g.TenantId == tenantID)
	})
}

// uniqueID checks the result of a list call and returns the ID of the only item that matches.
// It returns "" when no item matches, and an error when more than one matches,
// so that an ambiguous name never resolves to an arbitrary object.
// go-client always returns a non-nil response, so callers can pass resp fields before err is checked.
func uniqueID[T any](err error, status int, items []T, match func(T) (string, bool)) (string, error) {
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", status)
	}
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
