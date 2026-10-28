package config

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// ExternalNameConfigs contains all external name configurations for this
// provider.
//
// Do not use a "<parent_id>:<id>" template for application_oauth_scope,
// application_role, entity or entity_type_permission. Their Terraform read
// functions look up the bare object ID, and only the Terraform importers,
// which upjet does not call, split the composite form. With a template, the
// resource is not found after an import or a provider restart.
// fusionauth_registration can use one because its read ignores the ID.
var ExternalNameConfigs = map[string]config.ExternalName{
	"fusionauth_api_key":                      config.IdentifierFromProvider,
	"fusionauth_application_oauth_scope":      config.IdentifierFromProvider,
	"fusionauth_application_role":             config.IdentifierFromProvider,
	"fusionauth_application":                  config.IdentifierFromProvider,
	"fusionauth_consent":                      config.IdentifierFromProvider,
	"fusionauth_email":                        config.IdentifierFromProvider,
	"fusionauth_entity_grant":                 config.IdentifierFromProvider,
	"fusionauth_entity_type_permission":       config.IdentifierFromProvider,
	"fusionauth_entity_type":                  config.IdentifierFromProvider,
	"fusionauth_entity":                       config.IdentifierFromProvider,
	"fusionauth_form_field":                   config.IdentifierFromProvider,
	"fusionauth_form":                         config.IdentifierFromProvider,
	"fusionauth_generic_connector":            config.IdentifierFromProvider,
	"fusionauth_generic_messenger":            config.IdentifierFromProvider,
	"fusionauth_group":                        config.IdentifierFromProvider,
	"fusionauth_idp_apple":                    config.IdentifierFromProvider,
	"fusionauth_idp_external_jwt":             config.IdentifierFromProvider,
	"fusionauth_idp_facebook":                 config.IdentifierFromProvider,
	"fusionauth_idp_google":                   config.IdentifierFromProvider,
	"fusionauth_idp_linkedin":                 config.IdentifierFromProvider,
	"fusionauth_idp_open_id_connect":          config.IdentifierFromProvider,
	"fusionauth_idp_saml_v2_idp_initated":     config.IdentifierFromProvider,
	"fusionauth_idp_saml_v2":                  config.IdentifierFromProvider,
	"fusionauth_idp_sony_psn":                 config.IdentifierFromProvider,
	"fusionauth_idp_steam":                    config.IdentifierFromProvider,
	"fusionauth_idp_twitch":                   config.IdentifierFromProvider,
	"fusionauth_idp_xbox":                     config.IdentifierFromProvider,
	"fusionauth_imported_key":                 config.IdentifierFromProvider,
	"fusionauth_key":                          config.IdentifierFromProvider,
	"fusionauth_lambda":                       config.IdentifierFromProvider,
	"fusionauth_ldap_connector":               config.IdentifierFromProvider,
	"fusionauth_reactor":                      config.IdentifierFromProvider,
	"fusionauth_registration":                 config.TemplatedStringAsIdentifier("", "{{ .parameters.user_id }}:{{ .parameters.application_id }}"),
	"fusionauth_sms_message_template":         config.IdentifierFromProvider,
	"fusionauth_system_configuration":         config.IdentifierFromProvider,
	"fusionauth_tenant_manager_configuration": config.IdentifierFromProvider,
	"fusionauth_tenant":                       config.IdentifierFromProvider,
	"fusionauth_theme":                        config.IdentifierFromProvider,
	"fusionauth_twilio_messenger":             config.IdentifierFromProvider,
	"fusionauth_user_action":                  config.IdentifierFromProvider,
	"fusionauth_user_group_membership":        config.IdentifierFromProvider,
	"fusionauth_user":                         config.IdentifierFromProvider,
	"fusionauth_webhook":                      config.IdentifierFromProvider,
}

// ExternalNameConfigured returns the list of all resources with external name
// configurations.
func ExternalNameConfigured() []string {
	l := make([]string, 0, len(ExternalNameConfigs))
	for name := range ExternalNameConfigs {
		l = append(l, name)
	}
	return l
}

// ExternalNameConfigurations applies all external name configurations for each
// resource separately.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}
