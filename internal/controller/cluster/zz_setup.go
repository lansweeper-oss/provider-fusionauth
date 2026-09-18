// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	oauthscope "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/application/oauthscope"
	role "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/application/role"
	grant "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/entity/grant"
	field "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/form/field"
	apikey "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/apikey"
	application "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/application"
	consent "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/consent"
	email "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/email"
	entity "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/entity"
	entitytype "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/entitytype"
	entitytypepermission "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/entitytypepermission"
	form "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/form"
	group "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/group"
	key "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/key"
	lambda "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/lambda"
	reactor "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/reactor"
	registration "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/registration"
	tenant "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/tenant"
	theme "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/theme"
	user "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/user"
	webhook "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/fusionauth/webhook"
	connector "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/generic/connector"
	messenger "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/generic/messenger"
	apple "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/apple"
	externaljwt "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/externaljwt"
	facebook "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/facebook"
	google "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/google"
	linkedin "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/linkedin"
	openidconnect "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/openidconnect"
	samlv2 "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/samlv2"
	samlv2idpinitated "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/samlv2idpinitated"
	sonypsn "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/sonypsn"
	steam "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/steam"
	twitch "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/twitch"
	xbox "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/idp/xbox"
	keyimported "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/imported/key"
	connectorldap "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/ldap/connector"
	providerconfig "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/providerconfig"
	messagetemplate "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/sms/messagetemplate"
	configuration "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/system/configuration"
	managerconfiguration "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/tenant/managerconfiguration"
	messengertwilio "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/twilio/messenger"
	action "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/user/action"
	groupmembership "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster/user/groupmembership"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		oauthscope.Setup,
		role.Setup,
		grant.Setup,
		field.Setup,
		apikey.Setup,
		application.Setup,
		consent.Setup,
		email.Setup,
		entity.Setup,
		entitytype.Setup,
		entitytypepermission.Setup,
		form.Setup,
		group.Setup,
		key.Setup,
		lambda.Setup,
		reactor.Setup,
		registration.Setup,
		tenant.Setup,
		theme.Setup,
		user.Setup,
		webhook.Setup,
		connector.Setup,
		messenger.Setup,
		apple.Setup,
		externaljwt.Setup,
		facebook.Setup,
		google.Setup,
		linkedin.Setup,
		openidconnect.Setup,
		samlv2.Setup,
		samlv2idpinitated.Setup,
		sonypsn.Setup,
		steam.Setup,
		twitch.Setup,
		xbox.Setup,
		keyimported.Setup,
		connectorldap.Setup,
		providerconfig.Setup,
		messagetemplate.Setup,
		configuration.Setup,
		managerconfiguration.Setup,
		messengertwilio.Setup,
		action.Setup,
		groupmembership.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		oauthscope.SetupGated,
		role.SetupGated,
		grant.SetupGated,
		field.SetupGated,
		apikey.SetupGated,
		application.SetupGated,
		consent.SetupGated,
		email.SetupGated,
		entity.SetupGated,
		entitytype.SetupGated,
		entitytypepermission.SetupGated,
		form.SetupGated,
		group.SetupGated,
		key.SetupGated,
		lambda.SetupGated,
		reactor.SetupGated,
		registration.SetupGated,
		tenant.SetupGated,
		theme.SetupGated,
		user.SetupGated,
		webhook.SetupGated,
		connector.SetupGated,
		messenger.SetupGated,
		apple.SetupGated,
		externaljwt.SetupGated,
		facebook.SetupGated,
		google.SetupGated,
		linkedin.SetupGated,
		openidconnect.SetupGated,
		samlv2.SetupGated,
		samlv2idpinitated.SetupGated,
		sonypsn.SetupGated,
		steam.SetupGated,
		twitch.SetupGated,
		xbox.SetupGated,
		keyimported.SetupGated,
		connectorldap.SetupGated,
		providerconfig.SetupGated,
		messagetemplate.SetupGated,
		configuration.SetupGated,
		managerconfiguration.SetupGated,
		messengertwilio.SetupGated,
		action.SetupGated,
		groupmembership.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhookWithManager registers conversion webhooks for all resource kinds in the group.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		oauthscope.SetupWebhookWithManager,
		role.SetupWebhookWithManager,
		grant.SetupWebhookWithManager,
		field.SetupWebhookWithManager,
		apikey.SetupWebhookWithManager,
		application.SetupWebhookWithManager,
		consent.SetupWebhookWithManager,
		email.SetupWebhookWithManager,
		entity.SetupWebhookWithManager,
		entitytype.SetupWebhookWithManager,
		entitytypepermission.SetupWebhookWithManager,
		form.SetupWebhookWithManager,
		group.SetupWebhookWithManager,
		key.SetupWebhookWithManager,
		lambda.SetupWebhookWithManager,
		reactor.SetupWebhookWithManager,
		registration.SetupWebhookWithManager,
		tenant.SetupWebhookWithManager,
		theme.SetupWebhookWithManager,
		user.SetupWebhookWithManager,
		webhook.SetupWebhookWithManager,
		connector.SetupWebhookWithManager,
		messenger.SetupWebhookWithManager,
		apple.SetupWebhookWithManager,
		externaljwt.SetupWebhookWithManager,
		facebook.SetupWebhookWithManager,
		google.SetupWebhookWithManager,
		linkedin.SetupWebhookWithManager,
		openidconnect.SetupWebhookWithManager,
		samlv2.SetupWebhookWithManager,
		samlv2idpinitated.SetupWebhookWithManager,
		sonypsn.SetupWebhookWithManager,
		steam.SetupWebhookWithManager,
		twitch.SetupWebhookWithManager,
		xbox.SetupWebhookWithManager,
		keyimported.SetupWebhookWithManager,
		connectorldap.SetupWebhookWithManager,
		providerconfig.SetupWebhookWithManager,
		messagetemplate.SetupWebhookWithManager,
		configuration.SetupWebhookWithManager,
		managerconfiguration.SetupWebhookWithManager,
		messengertwilio.SetupWebhookWithManager,
		action.SetupWebhookWithManager,
		groupmembership.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
