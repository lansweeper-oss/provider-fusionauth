package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gofusionauth "github.com/FusionAuth/go-client/pkg/fusionauth"
	upstreamfusionauth "github.com/FusionAuth/terraform-provider-fusionauth/fusionauth"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUserID   = "6b2f3c1e-1d2a-4b5c-8d9e-0f1a2b3c4d5e"
	testTenantID = "0e1d2c3b-4a59-4687-9a1b-2c3d4e5f6a7b"
)

func TestReadUserByLoginID(t *testing.T) {
	t.Parallel()
	res := GetProvider().Resources["fusionauth_user"].TerraformResource
	cases := map[string]struct {
		externalName string
		loginIDFound bool
		wantID       string
		wantLookup   bool
	}{
		"UserIDSkipsLookup": {externalName: testUserID, wantID: testUserID},
		"EmailIsResolved":   {externalName: "jdoe@example.com", loginIDFound: true, wantID: testUserID, wantLookup: true},
		"UnknownLoginID":    {externalName: "nobody@example.com", wantID: "", wantLookup: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var lookedUp bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/api/user" && r.URL.Query().Get("loginId") != "":
					lookedUp = true
					assert.Equal(t, tc.externalName, r.URL.Query().Get("loginId"))
					assert.Equal(t, testTenantID, r.Header.Get("X-FusionAuth-TenantId"))
					if !tc.loginIDFound {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					writeUser(t, w)
				case r.URL.Path == "/api/user/"+testUserID:
					writeUser(t, w)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer srv.Close()

			u, err := url.Parse(srv.URL)
			require.NoError(t, err)
			meta := upstreamfusionauth.Client{FAClient: *gofusionauth.NewClient(srv.Client(), u, "key")}

			d := res.Data(&terraform.InstanceState{ID: tc.externalName, Attributes: map[string]string{"tenant_id": testTenantID}})

			diags := res.ReadContext(context.Background(), d, meta)
			require.False(t, diags.HasError(), "%v", diags)
			assert.Equal(t, tc.wantID, d.Id())
			assert.Equal(t, tc.wantLookup, lookedUp)
		})
	}
}

func writeUser(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"user": map[string]any{"id": testUserID, "email": "jdoe@example.com", "tenantId": testTenantID},
	}))
}

func TestLookupByName(t *testing.T) {
	t.Parallel()
	p := GetProvider()
	const (
		otherTenantID = "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"
		idA           = "11111111-1111-4111-8111-111111111111"
		idB           = "22222222-2222-4222-8222-222222222222"
	)
	obj := []map[string]any{{"id": idA, "name": "obj"}}
	lists := map[string]any{
		"/api/tenant": map[string]any{"tenants": []map[string]any{
			{"id": idA, "name": "Default"},
			{"id": idB, "name": "Default-2"},
		}},
		"/api/application": map[string]any{"applications": []map[string]any{
			{"id": idA, "name": "app", "tenantId": testTenantID},
			{"id": idB, "name": "app", "tenantId": otherTenantID},
		}},
		"/api/group": map[string]any{"groups": []map[string]any{
			{"id": idA, "name": "admins", "tenantId": testTenantID},
			{"id": idB, "name": "admins", "tenantId": otherTenantID},
		}},
		"/api/application/" + idA: map[string]any{"application": map[string]any{
			"id":     idA,
			"roles":  []map[string]any{{"id": idB, "name": "admin"}},
			"scopes": []map[string]any{{"id": idA, "name": "read"}},
		}},
		"/api/entity/type/" + idA: map[string]any{"entityType": map[string]any{
			"id": idA, "permissions": []map[string]any{{"id": idB, "name": "edit"}},
		}},
		"/api/lambda":         map[string]any{"lambdas": obj},
		"/api/key":            map[string]any{"keys": obj},
		"/api/theme":          map[string]any{"themes": obj},
		"/api/email/template": map[string]any{"emailTemplates": obj},
		"/api/form":           map[string]any{"forms": obj},
		"/api/consent":        map[string]any{"consents": obj},
		"/api/entity/type":    map[string]any{"entityTypes": obj},
		"/api/user-action":    map[string]any{"userActions": obj},
		"/api/connector": map[string]any{"connectors": []map[string]any{
			{"id": idA, "name": "same", "type": "LDAP"},
			{"id": idB, "name": "same", "type": "Generic"},
		}},
		"/api/messenger": map[string]any{"messengers": []map[string]any{
			{"id": idA, "name": "same", "type": "Twilio"},
			{"id": idB, "name": "same", "type": "Generic"},
		}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := lists[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(body))
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	meta := upstreamfusionauth.Client{FAClient: *gofusionauth.NewClient(srv.Client(), u, "key")}

	type tcase struct {
		resource string
		name     string
		attrs    map[string]string
		wantID   string
		wantErr  bool
	}
	inTenant := func(id string) map[string]string { return map[string]string{"tenant_id": id} }
	inApp := func(id string) map[string]string { return map[string]string{"application_id": id} }
	cases := map[string]tcase{
		"TenantExactName":             {resource: "fusionauth_tenant", name: "Default", wantID: idA},
		"TenantNoPartialMatch":        {resource: "fusionauth_tenant", name: "Defau", wantID: ""},
		"ApplicationInTenant":         {resource: "fusionauth_application", name: "app", attrs: inTenant(otherTenantID), wantID: idB},
		"ApplicationAmbiguous":        {resource: "fusionauth_application", name: "app", wantErr: true},
		"ApplicationUnknown":          {resource: "fusionauth_application", name: "nope", wantID: ""},
		"GroupInTenant":               {resource: "fusionauth_group", name: "admins", attrs: inTenant(testTenantID), wantID: idA},
		"GroupAmbiguousNoTenant":      {resource: "fusionauth_group", name: "admins", wantErr: true},
		"RoleInApplication":           {resource: "fusionauth_application_role", name: "admin", attrs: inApp(idA), wantID: idB},
		"RoleUnknownName":             {resource: "fusionauth_application_role", name: "nope", attrs: inApp(idA), wantID: ""},
		"RoleUnknownApplication":      {resource: "fusionauth_application_role", name: "admin", attrs: inApp(idB), wantID: ""},
		"RoleWithoutApplication":      {resource: "fusionauth_application_role", name: "admin", wantErr: true},
		"OAuthScopeInApplication":     {resource: "fusionauth_application_oauth_scope", name: "read", attrs: inApp(idA), wantID: idA},
		"OAuthScopeNoApplication":     {resource: "fusionauth_application_oauth_scope", name: "read", wantErr: true},
		"PermissionInEntityType":      {resource: "fusionauth_entity_type_permission", name: "edit", attrs: map[string]string{"entity_type_id": idA}, wantID: idB},
		"PermissionNoEntityType":      {resource: "fusionauth_entity_type_permission", name: "edit", wantErr: true},
		"GenericConnectorOnlyGeneric": {resource: "fusionauth_generic_connector", name: "same", wantID: idB},
		"LDAPConnectorOnlyLDAP":       {resource: "fusionauth_ldap_connector", name: "same", wantID: idA},
		"GenericMessengerOnlyGeneric": {resource: "fusionauth_generic_messenger", name: "same", wantID: idB},
		"TwilioMessengerOnlyTwilio":   {resource: "fusionauth_twilio_messenger", name: "same", wantID: idA},
	}
	for _, res := range []string{
		"fusionauth_lambda", "fusionauth_key", "fusionauth_theme", "fusionauth_email",
		"fusionauth_form", "fusionauth_consent", "fusionauth_entity_type", "fusionauth_user_action",
	} {
		cases[res] = tcase{resource: res, name: "obj", wantID: idA}
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			lookup, ok := nameLookups[tc.resource]
			require.True(t, ok, "no lookup for %s", tc.resource)
			res := p.Resources[tc.resource].TerraformResource
			d := res.Data(&terraform.InstanceState{ID: tc.name, Attributes: tc.attrs})

			id, err := lookup(context.Background(), d, meta, tc.name)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantID, id)
		})
	}
}

func TestNameLookupsAreProviderResources(t *testing.T) {
	t.Parallel()
	p := GetProvider()
	for name := range nameLookups {
		assert.Contains(t, p.Resources, name)
	}
}
