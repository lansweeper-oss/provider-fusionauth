package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	gofusionauth "github.com/FusionAuth/go-client/pkg/fusionauth"
	upstreamfusionauth "github.com/FusionAuth/terraform-provider-fusionauth/fusionauth"
	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/types/name"
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
	"fusionauth_tenant":                  listLookup{uri: "/api/tenant", path: "tenants"}.lookup,
	"fusionauth_application":             listLookup{uri: "/api/application", path: "applications", tenant: true}.lookup,
	"fusionauth_group":                   listLookup{uri: "/api/group", path: "groups", tenant: true}.lookup,
	"fusionauth_application_role":        listLookup{uri: "/api/application/%s", path: "application.roles", parent: "application_id"}.lookup,
	"fusionauth_application_oauth_scope": listLookup{uri: "/api/application/%s", path: "application.scopes", parent: "application_id"}.lookup,
	"fusionauth_entity_type_permission":  listLookup{uri: "/api/entity/type/%s", path: "entityType.permissions", parent: "entity_type_id"}.lookup,
	"fusionauth_lambda":                  listLookup{uri: "/api/lambda", path: "lambdas"}.lookup,
	"fusionauth_key":                     listLookup{uri: "/api/key", path: "keys"}.lookup,
	"fusionauth_theme":                   listLookup{uri: "/api/theme", path: "themes"}.lookup,
	"fusionauth_email":                   listLookup{uri: "/api/email/template", path: "emailTemplates"}.lookup,
	"fusionauth_form":                    listLookup{uri: "/api/form", path: "forms"}.lookup,
	"fusionauth_consent":                 listLookup{uri: "/api/consent", path: "consents"}.lookup,
	"fusionauth_entity_type":             listLookup{uri: "/api/entity/type", path: "entityTypes"}.lookup,
	"fusionauth_user_action":             listLookup{uri: "/api/user-action", path: "userActions"}.lookup,
	"fusionauth_generic_connector":       listLookup{uri: "/api/connector", path: "connectors", typ: gofusionauth.ConnectorType_Generic.String()}.lookup,
	"fusionauth_ldap_connector":          listLookup{uri: "/api/connector", path: "connectors", typ: gofusionauth.ConnectorType_LDAP.String()}.lookup,
	"fusionauth_generic_messenger":       listLookup{uri: "/api/messenger", path: "messengers", typ: gofusionauth.MessengerType_Generic.String()}.lookup,
	"fusionauth_twilio_messenger":        listLookup{uri: "/api/messenger", path: "messengers", typ: gofusionauth.MessengerType_Twilio.String()}.lookup,
}

// configureLookups wraps the read of every resource in nameLookups with readWithLookup.
func configureLookups(p *config.Provider) {
	for resource, lookup := range nameLookups {
		p.AddResourceConfigurator(resource, func(r *config.Resource) {
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

// listLookup resolves a name by listing objects and matching the name exactly.
type listLookup struct {
	uri    string // list endpoint, with %s for the parent ID of a child object
	path   string // dot-separated JSON keys that lead to the list in the response
	parent string // Terraform field with the parent ID; required for child objects
	tenant bool   // match within tenant_id when it is set
	typ    string // only match objects of this type
}

// listResponse keeps the decoded body of a list call. go-client sets the status code.
type listResponse struct {
	gofusionauth.BaseHTTPResponse
	body map[string]any
}

func (r *listResponse) UnmarshalJSON(b []byte) error { return json.Unmarshal(b, &r.body) }

// lookup returns the ID of the only object that matches. It returns "" when no
// object matches or the parent does not exist, and an error when more than one
// matches, so that an ambiguous name never resolves to an arbitrary object.
func (l listLookup) lookup(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client, key string) (string, error) {
	items, err := l.list(ctx, d, c)
	if err != nil {
		return "", err
	}
	tenantID := ""
	if l.tenant {
		tenantID, _ = d.Get("tenant_id").(string)
	}

	var found []string
	for _, item := range items {
		obj, _ := item.(map[string]any)
		if l.matches(obj, key, tenantID) {
			found = append(found, str(obj, "id"))
		}
	}
	switch len(found) {
	case 0:
		return "", nil
	case 1:
		return found[0], nil
	}
	hint := "use the ID instead"
	if l.tenant {
		hint += ", or set tenantId"
	}
	return "", fmt.Errorf("name matches %d objects %v: %s", len(found), found, hint)
}

// matches reports whether obj has the name key, within tenantID and of type l.typ when they are set.
func (l listLookup) matches(obj map[string]any, key, tenantID string) bool {
	return str(obj, "name") == key &&
		(tenantID == "" || str(obj, "tenantId") == tenantID) &&
		(l.typ == "" || str(obj, "type") == l.typ)
}

// str returns the string field k of obj, or "" when it is missing.
func str(obj map[string]any, k string) string {
	v, _ := obj[k].(string)
	return v
}

// list returns the objects at l.path in the list response. It returns no
// objects when the parent does not exist.
func (l listLookup) list(ctx context.Context, d *schema.ResourceData, c upstreamfusionauth.Client) ([]any, error) {
	uri := l.uri
	if l.parent != "" {
		parentID, _ := d.Get(l.parent).(string)
		if parentID == "" {
			return nil, fmt.Errorf("set %s to import by name", name.NewFromSnake(l.parent).LowerCamelComputed)
		}
		uri = fmt.Sprintf(l.uri, url.PathEscape(parentID))
	}

	var resp listResponse
	if err := c.FAClient.Start(&resp, nil).WithUri(uri).WithMethod(http.MethodGet).Do(ctx); err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var v any = resp.body
	for k := range strings.SplitSeq(l.path, ".") {
		m, _ := v.(map[string]any)
		v = m[k]
	}
	items, _ := v.([]any)
	return items, nil
}
