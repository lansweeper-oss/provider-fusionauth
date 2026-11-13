package config

import (
	"context"
	"fmt"
	"net/http"

	gofusionauth "github.com/FusionAuth/go-client/pkg/fusionauth"
	upstreamfusionauth "github.com/FusionAuth/terraform-provider-fusionauth/fusionauth"
	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// lookupFn resolves a human-readable key (a name, an email) to the object ID.
// It returns "" when no object matches.
type lookupFn func(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, key string) (string, error)

// nameLookups lists the resources whose external name can be a name instead of the object ID.
var nameLookups = map[string]lookupFn{
	"fusionauth_user":                    lookupUserByLoginID,
	"fusionauth_tenant":                  lookupTenantByName,
	"fusionauth_application":             lookupApplicationByName,
	"fusionauth_group":                   lookupGroupByName,
	"fusionauth_application_role":        lookupApplicationRoleByName,
	"fusionauth_application_oauth_scope": lookupOAuthScopeByName,
	"fusionauth_entity_type_permission":  lookupEntityTypePermissionByName,
	"fusionauth_lambda":                  lookupLambdaByName,
	"fusionauth_key":                     lookupKeyByName,
	"fusionauth_theme":                   lookupThemeByName,
	"fusionauth_email":                   lookupEmailTemplateByName,
	"fusionauth_form":                    lookupFormByName,
	"fusionauth_consent":                 lookupConsentByName,
	"fusionauth_entity_type":             lookupEntityTypeByName,
	"fusionauth_user_action":             lookupUserActionByName,
	"fusionauth_generic_connector":       lookupConnectorByName(gofusionauth.ConnectorType_Generic),
	"fusionauth_ldap_connector":          lookupConnectorByName(gofusionauth.ConnectorType_LDAP),
	"fusionauth_generic_messenger":       lookupMessengerByName(gofusionauth.MessengerType_Generic),
	"fusionauth_twilio_messenger":        lookupMessengerByName(gofusionauth.MessengerType_Twilio),
}

// configureLookups wraps the read of every resource in nameLookups with readWithLookup.
func configureLookups(p *config.Provider) {
	for name, lookup := range nameLookups {
		p.AddResourceConfigurator(name, func(r *config.Resource) {
			r.TerraformResource.ReadContext = readWithLookup(r.TerraformResource.ReadContext, lookup)
		})
	}
}

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

// lookupApplicationRoleByName resolves a role name to the role ID.
// A role name is unique within its application.
func lookupApplicationRoleByName(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	appID, err := parentID(d, "application_id", "applicationId")
	if err != nil {
		return "", err
	}
	resp, err := c.FAClient.RetrieveApplicationWithContext(ctx, appID)
	return uniqueID(err, resp.StatusCode, resp.Application.Roles, func(r gofusionauth.ApplicationRole) (string, bool) {
		return r.Id, r.Name == name
	})
}

// lookupOAuthScopeByName resolves an OAuth scope name to the scope ID.
// A scope name is unique within its application.
func lookupOAuthScopeByName(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	appID, err := parentID(d, "application_id", "applicationId")
	if err != nil {
		return "", err
	}
	resp, err := c.FAClient.RetrieveApplicationWithContext(ctx, appID)
	return uniqueID(err, resp.StatusCode, resp.Application.Scopes, func(s gofusionauth.ApplicationOAuthScope) (string, bool) {
		return s.Id, s.Name == name
	})
}

// lookupEntityTypePermissionByName resolves a permission name to the permission ID.
// A permission name is unique within its entity type.
func lookupEntityTypePermissionByName(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	entityTypeID, err := parentID(d, "entity_type_id", "entityTypeId")
	if err != nil {
		return "", err
	}
	resp, _, err := c.FAClient.RetrieveEntityTypeWithContext(ctx, entityTypeID)
	return uniqueID(err, resp.StatusCode, resp.EntityType.Permissions, func(p gofusionauth.EntityTypePermission) (string, bool) {
		return p.Id, p.Name == name
	})
}

// parentID returns the ID of the parent that a child name is matched within.
func parentID(d *schema.ResourceData, field, param string) (string, error) {
	id, _ := d.Get(field).(string)
	if id == "" {
		return "", fmt.Errorf("set %s to import by name", param)
	}
	return id, nil
}

func lookupLambdaByName(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, err := c.FAClient.RetrieveLambdasWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.Lambdas, func(l gofusionauth.Lambda) (string, bool) {
		return l.Id, l.Name == name
	})
}

func lookupKeyByName(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, err := c.FAClient.RetrieveKeysWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.Keys, func(k gofusionauth.Key) (string, bool) {
		return k.Id, k.Name == name
	})
}

func lookupThemeByName(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, err := c.FAClient.RetrieveThemesWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.Themes, func(t gofusionauth.Theme) (string, bool) {
		return t.Id, t.Name == name
	})
}

func lookupEmailTemplateByName(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, err := c.FAClient.RetrieveEmailTemplatesWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.EmailTemplates, func(e gofusionauth.EmailTemplate) (string, bool) {
		return e.Id, e.Name == name
	})
}

func lookupFormByName(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, err := c.FAClient.RetrieveFormsWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.Forms, func(f gofusionauth.Form) (string, bool) {
		return f.Id, f.Name == name
	})
}

func lookupConsentByName(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, err := c.FAClient.RetrieveConsentsWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.Consents, func(x gofusionauth.Consent) (string, bool) {
		return x.Id, x.Name == name
	})
}

func lookupEntityTypeByName(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, _, err := c.FAClient.RetrieveEntityTypesWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.EntityTypes, func(e gofusionauth.EntityType) (string, bool) {
		return e.Id, e.Name == name
	})
}

func lookupUserActionByName(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
	resp, err := c.FAClient.RetrieveUserActionsWithContext(ctx)
	return uniqueID(err, resp.StatusCode, resp.UserActions, func(a gofusionauth.UserAction) (string, bool) {
		return a.Id, a.Name == name
	})
}

// lookupConnectorByName matches only connectors of type t, so that a
// GenericConnector never resolves to an LDAP connector.
func lookupConnectorByName(t gofusionauth.ConnectorType) lookupFn {
	return func(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
		resp, err := c.FAClient.RetrieveConnectorsWithContext(ctx)
		return uniqueID(err, resp.StatusCode, resp.Connectors, func(x gofusionauth.BaseConnectorConfiguration) (string, bool) {
			return x.Id, x.Type == t && x.Name == name
		})
	}
}

// lookupMessengerByName matches only messengers of type t.
func lookupMessengerByName(t gofusionauth.MessengerType) lookupFn {
	return func(ctx context.Context, _ *schema.ResourceData, c upstreamfusionauth.Client, name string) (string, error) {
		resp, err := c.FAClient.RetrieveMessengersWithContext(ctx)
		return uniqueID(err, resp.StatusCode, resp.Messengers, func(x gofusionauth.BaseMessengerConfiguration) (string, bool) {
			return x.Id, x.Type == t && x.Name == name
		})
	}
}

// uniqueID checks the result of a list call and returns the ID of the only item that matches.
// It returns "" when no item matches or the parent does not exist, and an error when more than one matches,
// so that an ambiguous name never resolves to an arbitrary object.
// go-client always returns a non-nil response, so callers can pass resp fields before err is checked.
func uniqueID[T any](err error, status int, items []T, match func(T) (string, bool)) (string, error) {
	if err != nil {
		return "", err
	}
	if status == http.StatusNotFound {
		// The parent of a child object does not exist.
		return "", nil
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
		return "", fmt.Errorf("name matches %d objects %v: use the ID instead, or set tenantId if the kind has one", len(found), found)
	}
	if len(found) == 0 {
		return "", nil
	}
	return found[0], nil
}
