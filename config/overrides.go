package config

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

var (
	applicationRef = config.Reference{TerraformName: "fusionauth_application"}
	consentRef     = config.Reference{TerraformName: "fusionauth_consent"}
	emailRef       = config.Reference{TerraformName: "fusionauth_email"}
	entityRef      = config.Reference{TerraformName: "fusionauth_entity"}
	entityTypeRef  = config.Reference{TerraformName: "fusionauth_entity_type"}
	formRef        = config.Reference{TerraformName: "fusionauth_form"}
	groupRef       = config.Reference{TerraformName: "fusionauth_group"}
	keyRef         = config.Reference{TerraformName: "fusionauth_key"}
	lambdaRef      = config.Reference{TerraformName: "fusionauth_lambda"}
	smsTemplateRef = config.Reference{TerraformName: "fusionauth_sms_message_template"}
	tenantRef      = config.Reference{TerraformName: "fusionauth_tenant"}
	themeRef       = config.Reference{TerraformName: "fusionauth_theme"}
	userRef        = config.Reference{TerraformName: "fusionauth_user"}
)

// Configure adds resource-specific overrides to the provider configuration.
func Configure(p *config.Provider) {
	// --- Flat resources (ShortGroup="") ---
	// Single-segment terraform names that would otherwise get ShortGroup="fusionauth"
	// producing the redundant "fusionauth.fusionauth.crossplane.io" group.

	p.AddResourceConfigurator("fusionauth_api_key", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "APIKey"
		r.References["tenant_id"] = tenantRef
	})

	p.AddResourceConfigurator("fusionauth_application", func(r *config.Resource) {
		r.ShortGroup = ""
		r.LateInitializer = config.LateInitializer{
			IgnoredFields: []string{"multi_factor_configuration"},
		}
		r.References["jwt_configuration.access_token_id"] = keyRef
		r.References["jwt_configuration.id_token_key_id"] = keyRef
		r.References["lambda_configuration.access_token_populate_id"] = lambdaRef
		r.References["lambda_configuration.id_token_populate_id"] = lambdaRef
		r.References["lambda_configuration.multi_factor_requirement_id"] = lambdaRef
		r.References["lambda_configuration.samlv2_populate_id"] = lambdaRef
		r.References["lambda_configuration.self_service_registration_validation_id"] = lambdaRef
		r.References["lambda_configuration.userinfo_populate_id"] = lambdaRef
		r.References["phone_configuration.forgot_password_template_id"] = smsTemplateRef
		r.References["phone_configuration.identity_update_template_id"] = smsTemplateRef
		r.References["phone_configuration.login_id_in_use_on_create_template_id"] = smsTemplateRef
		r.References["phone_configuration.login_id_in_use_on_update_template_id"] = smsTemplateRef
		r.References["phone_configuration.login_new_device_template_id"] = smsTemplateRef
		r.References["phone_configuration.login_suspicious_template_id"] = smsTemplateRef
		r.References["phone_configuration.password_reset_success_template_id"] = smsTemplateRef
		r.References["phone_configuration.password_update_template_id"] = smsTemplateRef
		r.References["phone_configuration.passwordless_template_id"] = smsTemplateRef
		r.References["phone_configuration.set_password_template_id"] = smsTemplateRef
		r.References["phone_configuration.two_factor_method_add_template_id"] = smsTemplateRef
		r.References["phone_configuration.two_factor_method_remove_template_id"] = smsTemplateRef
		r.References["phone_configuration.verification_complete_template_id"] = smsTemplateRef
		r.References["phone_configuration.verification_template_id"] = smsTemplateRef
		r.References["registration_configuration.form_id"] = formRef
		r.References["tenant_id"] = tenantRef
		r.References["theme_id"] = themeRef
		r.References["verification_email_template_id"] = emailRef
	})

	p.AddResourceConfigurator("fusionauth_consent", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Consent"
		r.References["consent_email_template_id"] = emailRef
	})

	p.AddResourceConfigurator("fusionauth_email", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Email"
	})

	p.AddResourceConfigurator("fusionauth_form", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Form"
	})

	p.AddResourceConfigurator("fusionauth_group", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Group"
		r.References["tenant_id"] = tenantRef
	})

	p.AddResourceConfigurator("fusionauth_key", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Key"
	})

	p.AddResourceConfigurator("fusionauth_lambda", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Lambda"
	})

	p.AddResourceConfigurator("fusionauth_reactor", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Reactor"
	})

	p.AddResourceConfigurator("fusionauth_registration", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Registration"
		r.References["application_id"] = applicationRef
		r.References["user_id"] = userRef
	})

	p.AddResourceConfigurator("fusionauth_theme", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Theme"
		r.References["source_theme_id"] = themeRef
	})

	p.AddResourceConfigurator("fusionauth_user", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "User"
		r.References["tenant_id"] = tenantRef
	})

	p.AddResourceConfigurator("fusionauth_webhook", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Webhook"
		r.References["signature_configuration.signing_key_id"] = keyRef
		r.References["ssl_certificate_key_id"] = keyRef
	})

	configureTenant(p)

	// --- Inferred group resources (keep upjet default ShortGroup) ---

	// application.* group
	p.AddResourceConfigurator("fusionauth_application_role", func(r *config.Resource) {
		r.References["application_id"] = applicationRef
	})

	p.AddResourceConfigurator("fusionauth_application_oauth_scope", func(r *config.Resource) {
		r.References["application_id"] = applicationRef
	})

	// entity.* group — Kind overrides to avoid "type" reserved word
	p.AddResourceConfigurator("fusionauth_entity", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Entity"
		r.References["entity_type_id"] = entityTypeRef
		r.References["tenant_id"] = tenantRef
	})

	p.AddResourceConfigurator("fusionauth_entity_grant", func(r *config.Resource) {
		r.References["entity_id"] = entityRef
		r.References["recipient_entity_id"] = entityRef
		r.References["tenant_id"] = tenantRef
		r.References["user_id"] = userRef
	})

	p.AddResourceConfigurator("fusionauth_entity_type", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "EntityType"
		r.References["jwt_configuration.access_token_key_id"] = keyRef
	})

	p.AddResourceConfigurator("fusionauth_entity_type_permission", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "EntityTypePermission"
		r.References["entity_type_id"] = entityTypeRef
	})

	// form.* group
	p.AddResourceConfigurator("fusionauth_form_field", func(r *config.Resource) {
		r.References["consent_id"] = consentRef
	})

	// generic.* group
	p.AddResourceConfigurator("fusionauth_generic_connector", func(r *config.Resource) {
		r.References["ssl_certificate_key_id"] = keyRef
	})

	p.AddResourceConfigurator("fusionauth_generic_messenger", func(r *config.Resource) {
	})

	// idp.* group
	configureIDPApple(p)
	configureIDPExternalJWT(p)
	configureIDPFacebook(p)
	configureIDPGoogle(p)
	configureIDPLinkedIn(p)
	configureIDPOpenIDConnect(p)
	configureIDPSAMLv2(p)
	configureIDPSAMLv2IDPInitiated(p)
	configureIDPSonyPSN(p)
	configureIDPSteam(p)
	configureIDPTwitch(p)
	configureIDPXbox(p)

	// imported.* group
	p.AddResourceConfigurator("fusionauth_imported_key", func(r *config.Resource) {
	})

	// ldap.* group
	p.AddResourceConfigurator("fusionauth_ldap_connector", func(r *config.Resource) {
		r.References["lambda_configuration.reconcile_id"] = lambdaRef
	})

	// sms.* group
	p.AddResourceConfigurator("fusionauth_sms_message_template", func(r *config.Resource) {
	})

	// system.* group
	p.AddResourceConfigurator("fusionauth_system_configuration", func(r *config.Resource) {
	})

	// tenant.* group
	p.AddResourceConfigurator("fusionauth_tenant_manager_configuration", func(r *config.Resource) {
		r.References["application_configurations.application_id"] = applicationRef
		r.References["attribute_form_id"] = formRef
	})

	// twilio.* group
	p.AddResourceConfigurator("fusionauth_twilio_messenger", func(r *config.Resource) {
	})

	// user.* group
	p.AddResourceConfigurator("fusionauth_user_action", func(r *config.Resource) {
		r.References["cancel_email_template_id"] = emailRef
		r.References["end_email_template_id"] = emailRef
		r.References["modify_email_template_id"] = emailRef
		r.References["start_email_template_id"] = emailRef
	})

	p.AddResourceConfigurator("fusionauth_user_group_membership", func(r *config.Resource) {
		r.References["group_id"] = groupRef
		r.References["user_id"] = userRef
	})
}

func configureIDPCommon(r *config.Resource) {
	r.References["application_configuration.application_id"] = applicationRef
	r.References["lambda_reconcile_id"] = lambdaRef
	r.References["tenant_configuration.tenant_id"] = tenantRef
	r.References["tenant_id"] = tenantRef
}

func configureIDPApple(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_apple", func(r *config.Resource) {
		configureIDPCommon(r)
		r.References["application_configuration.key_id"] = keyRef
		r.References["key_id"] = keyRef
	})
}

func configureIDPExternalJWT(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_external_jwt", func(r *config.Resource) {
		configureIDPCommon(r)
		r.References["default_key_id"] = keyRef
	})
}

func configureIDPFacebook(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_facebook", func(r *config.Resource) {
		configureIDPCommon(r)
	})
}

func configureIDPGoogle(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_google", func(r *config.Resource) {
		configureIDPCommon(r)
	})
}

func configureIDPLinkedIn(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_linkedin", func(r *config.Resource) {
		configureIDPCommon(r)
	})
}

func configureIDPOpenIDConnect(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_open_id_connect", func(r *config.Resource) {
		configureIDPCommon(r)
	})
}

func configureIDPSAMLv2(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_saml_v2", func(r *config.Resource) {
		configureIDPCommon(r)
		r.References["assertion_configuration.decryption.key_transport_decryption_key_id"] = keyRef
		r.References["key_id"] = keyRef
	})
}

func configureIDPSAMLv2IDPInitiated(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_saml_v2_idp_initated", func(r *config.Resource) {
		configureIDPCommon(r)
		r.References["assertion_configuration.decryption.key_transport_decryption_key_id"] = keyRef
		r.References["key_id"] = keyRef
	})
}

func configureIDPSonyPSN(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_sony_psn", func(r *config.Resource) {
		configureIDPCommon(r)
	})
}

func configureIDPSteam(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_steam", func(r *config.Resource) {
		configureIDPCommon(r)
	})
}

func configureIDPTwitch(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_twitch", func(r *config.Resource) {
		configureIDPCommon(r)
	})
}

func configureIDPXbox(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_idp_xbox", func(r *config.Resource) {
		configureIDPCommon(r)
	})
}

func configureTenant(p *config.Provider) {
	p.AddResourceConfigurator("fusionauth_tenant", func(r *config.Resource) {
		r.ShortGroup = ""
		r.Kind = "Tenant"
		r.References["theme_id"] = themeRef
		// Skipped: failed_authentication_configuration.user_action_id ref to user_action
		// would create import cycle between fusionauth/ and user/ packages.
		r.References["email_configuration.admin_two_factor_method_remove_email_template_id"] = emailRef
		r.References["email_configuration.email_update_email_template_id"] = emailRef
		r.References["email_configuration.email_verified_email_template_id"] = emailRef
		r.References["email_configuration.forgot_password_email_template_id"] = emailRef
		r.References["email_configuration.login_id_in_use_on_create_email_template_id"] = emailRef
		r.References["email_configuration.login_id_in_use_on_update_email_template_id"] = emailRef
		r.References["email_configuration.login_new_device_email_template_id"] = emailRef
		r.References["email_configuration.login_suspicious_email_template_id"] = emailRef
		r.References["email_configuration.password_reset_success_email_template_id"] = emailRef
		r.References["email_configuration.password_update_email_template_id"] = emailRef
		r.References["email_configuration.passwordless_email_template_id"] = emailRef
		r.References["email_configuration.set_password_email_template_id"] = emailRef
		r.References["email_configuration.two_factor_method_add_email_template_id"] = emailRef
		r.References["email_configuration.two_factor_method_remove_email_template_id"] = emailRef
		r.References["email_configuration.verification_email_template_id"] = emailRef
		r.References["family_configuration.confirm_child_email_template_id"] = emailRef
		r.References["family_configuration.family_request_email_template_id"] = emailRef
		r.References["family_configuration.parent_registration_email_template_id"] = emailRef
		r.References["jwt_configuration.access_token_key_id"] = keyRef
		r.References["jwt_configuration.id_token_key_id"] = keyRef
		r.References["lambda_configuration.login_validation_id"] = lambdaRef
		r.References["lambda_configuration.multi_factor_requirement_id"] = lambdaRef
		r.References["lambda_configuration.scim_enterprise_user_request_converter_id"] = lambdaRef
		r.References["lambda_configuration.scim_enterprise_user_response_converter_id"] = lambdaRef
		r.References["lambda_configuration.scim_group_request_converter_id"] = lambdaRef
		r.References["lambda_configuration.scim_group_response_converter_id"] = lambdaRef
		r.References["lambda_configuration.scim_user_request_converter_id"] = lambdaRef
		r.References["lambda_configuration.scim_user_response_converter_id"] = lambdaRef
		r.References["oauth_configuration.client_credentials_access_token_populate_lambda_id"] = lambdaRef
		r.References["phone_configuration.admin_two_factor_method_remove_template_id"] = smsTemplateRef
		r.References["phone_configuration.forgot_password_template_id"] = smsTemplateRef
		r.References["phone_configuration.identity_update_template_id"] = smsTemplateRef
		r.References["phone_configuration.login_id_in_use_on_create_template_id"] = smsTemplateRef
		r.References["phone_configuration.login_id_in_use_on_update_template_id"] = smsTemplateRef
		r.References["phone_configuration.login_new_device_template_id"] = smsTemplateRef
		r.References["phone_configuration.login_suspicious_template_id"] = smsTemplateRef
		r.References["phone_configuration.password_reset_success_template_id"] = smsTemplateRef
		r.References["phone_configuration.password_update_template_id"] = smsTemplateRef
		r.References["phone_configuration.passwordless_template_id"] = smsTemplateRef
		r.References["phone_configuration.set_password_template_id"] = smsTemplateRef
		r.References["phone_configuration.two_factor_method_add_template_id"] = smsTemplateRef
		r.References["phone_configuration.two_factor_method_remove_template_id"] = smsTemplateRef
		r.References["phone_configuration.verification_complete_template_id"] = smsTemplateRef
		r.References["phone_configuration.verification_template_id"] = smsTemplateRef
		r.References["scim_server_configuration.client_entity_type_id"] = entityTypeRef
		r.References["scim_server_configuration.server_entity_type_id"] = entityTypeRef
	})
}
