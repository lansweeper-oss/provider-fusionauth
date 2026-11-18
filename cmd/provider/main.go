package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	changelogsv1alpha1 "github.com/crossplane/crossplane-runtime/v2/apis/changelogs/proto/v1alpha1"
	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/gate"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/customresourcesgate"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	tjcontroller "github.com/crossplane/upjet/v2/pkg/controller"
	"github.com/crossplane/upjet/v2/pkg/diffserver"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	apisCluster "github.com/lansweeper-oss/provider-fusionauth/apis/cluster"
	apisNamespaced "github.com/lansweeper-oss/provider-fusionauth/apis/namespaced"
	"github.com/lansweeper-oss/provider-fusionauth/config"
	"github.com/lansweeper-oss/provider-fusionauth/internal/clients"
	controllerCluster "github.com/lansweeper-oss/provider-fusionauth/internal/controller/cluster"
	controllerNamespaced "github.com/lansweeper-oss/provider-fusionauth/internal/controller/namespaced"
	"github.com/lansweeper-oss/provider-fusionauth/internal/features"
	"github.com/lansweeper-oss/provider-fusionauth/internal/version"
)

const (
	webhookTLSCertDirEnvVar = "WEBHOOK_TLS_CERT_DIR"
	tlsServerCertDirEnvVar  = "TLS_SERVER_CERTS_DIR"
	certsDirEnvVar          = "CERTS_DIR"
	tlsServerCertDir        = "/tls/server"

	// The diff server command, as kong's Context.Command reports it.
	cmdDiffServer = "internal diff-server"
)

type certsDir string

func (d certsDir) BeforeApply(certsDirSet *bool) error {
	// we record whether the command-line option "--certs-dir" was supplied
	*certsDirSet = true
	return nil
}

// startCmd runs the provider's controllers. It is the default command, so
// that the provider keeps behaving as it did before the internal commands
// were introduced when it is invoked with flags alone.
type startCmd struct {
	LeaderElection bool `help:"Use leader election for the controller manager" short:"l" default:"false" env:"LEADER_ELECTION"`

	SyncPeriod              time.Duration `name:"sync" help:"Controller manager sync period such as 300ms, 1.5h, or 2h45m" short:"s" default:"1h"`
	PollInterval            time.Duration `name:"poll" help:"Poll interval controls how often an individual resource should be checked for drift." default:"10m"`
	PollStateMetricInterval time.Duration `name:"poll-state-metric" help:"State metric recording interval" default:"5s"`

	MaxReconcileRate         int    `help:"The global maximum rate per second at which resources may be checked for drift from the desired state." default:"10"`
	EnableSecretCache        bool   `help:"Cache Secrets in memory. Disabling it reads Secrets directly from the API server, which saves memory on clusters with many Secrets at the cost of more API requests." default:"true" env:"ENABLE_SECRET_CACHE"`
	EnableManagementPolicies bool   `help:"Enable support for Management Policies" default:"true" env:"ENABLE_MANAGEMENT_POLICIES"`
	EnableChangeLogs         bool   `name:"enable-changelogs" help:"Enable support for capturing change logs during reconciliation" default:"false" env:"ENABLE_CHANGE_LOGS"`
	ChangelogsSocketPath     string `help:"Path for changelogs socket (if enabled)" default:"/var/run/changelogs/changelogs.sock" env:"CHANGELOGS_SOCKET_PATH"`

	WebhookPort        int      `help:"The port the webhook listens on" default:"9443" env:"WEBHOOK_PORT"`
	MetricsBindAddress string   `help:"The address the metrics server listens on" default:":8080" env:"METRICS_BIND_ADDRESS"`
	CertsDir           certsDir `help:"The directory that contains the server key and certificate" default:"${defaultCertsDir}" env:"${defaultCertsDirEnvVar}"`
}

// diffServerCmd serves upjet's diff gRPC service, which the Crossplane CLI
// drives to simulate changes. It never contacts FusionAuth or a cluster:
// plans are computed offline from the resources in each request.
type diffServerCmd struct {
	Network string `help:"The network the diff gRPC server listens on." default:"tcp" enum:"tcp,unix" env:"DIFF_SERVER_NETWORK"`
	Address string `help:"The address the diff gRPC server listens on. A socket path when the network is unix." default:":9099" env:"DIFF_SERVER_ADDRESS"`
}

var cli struct {
	Debug bool `help:"Run with debug logging" short:"d"`

	Start    startCmd `cmd:"" default:"withargs" help:"Start the provider's controllers."`
	Internal struct {
		DiffServer diffServerCmd `cmd:"" name:"diff-server" help:"Start a gRPC server serving the provider's diff services. It never contacts FusionAuth or a cluster: plans are computed offline from the resources in each request."`
	} `cmd:"" help:"Commands used internally by the Crossplane ecosystem. No compatibility guarantees are made for them."`
}

func main() { //nolint:gocyclo // linear flag wiring plus the diff-server dispatch
	certsDirSet := false
	ctx := kong.Parse(&cli,
		kong.Description("Crossplane Provider for fusionauth"),
		kong.Bind(&certsDirSet),
		kong.Vars{
			"defaultCertsDir":       tlsServerCertDir,
			"defaultCertsDirEnvVar": certsDirEnvVar,
		},
	)

	zl := zap.New(zap.UseDevMode(cli.Debug))
	log := logging.NewLogrLogger(zl.WithName("provider-fusionauth"))
	if cli.Debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	} else {
		// controller-runtime v0.23+ requires SetLogger to be called, otherwise
		// background goroutines (e.g. priority queue) will warn about missing logger.
		ctrl.SetLogger(logr.Discard())
	}

	// Dispatched before ctrl.GetConfig and the manager below: the diff server
	// plans from the resources in each request and never talks to a cluster or
	// to FusionAuth. Any other command is "start".
	if ctx.Command() == cmdDiffServer {
		ctx.FatalIfErrorf(serveDiffServer(ctrl.SetupSignalHandler(), log, cli.Internal.DiffServer.Network, cli.Internal.DiffServer.Address), "Cannot run the diff gRPC server")
		return
	}

	// currently, we configure the jitter to be the 5% of the poll interval
	pollJitter := time.Duration(float64(cli.Start.PollInterval) * 0.05)

	log.Debug("Starting", "sync-period", cli.Start.SyncPeriod.String(), "poll-interval", cli.Start.PollInterval.String(), "max-reconcile-rate", cli.Start.MaxReconcileRate)

	cfg, err := ctrl.GetConfig()
	ctx.FatalIfErrorf(err, "Cannot get API server rest config")

	// Get the TLS certs directory from the environment variables set by
	// Crossplane if they're available.
	// In older XP versions we used WEBHOOK_TLS_CERT_DIR, in newer versions
	// we use TLS_SERVER_CERTS_DIR. If an explicit certs dir is not supplied
	// via the command-line options, then these environment variables are used
	// instead.
	if !certsDirSet {
		// backwards-compatibility concerns
		xpCertsDir := os.Getenv(certsDirEnvVar)
		if xpCertsDir == "" {
			xpCertsDir = os.Getenv(tlsServerCertDirEnvVar)
		}
		if xpCertsDir == "" {
			xpCertsDir = os.Getenv(webhookTLSCertDirEnvVar)
		}
		// we probably don't need this condition but just to be on the
		// safe side, if we are missing any kong machinery details...
		if xpCertsDir != "" {
			cli.Start.CertsDir = certsDir(xpCertsDir)
		}
	}

	scheme := runtime.NewScheme()
	ctx.FatalIfErrorf(clientgoscheme.AddToScheme(scheme), "Cannot add client-go APIs to scheme")
	ctx.FatalIfErrorf(apisCluster.AddToScheme(scheme), "Cannot add cluster-scoped provider-fusionauth APIs to scheme")
	ctx.FatalIfErrorf(apisNamespaced.AddToScheme(scheme), "Cannot add namespaced provider-fusionauth APIs to scheme")
	ctx.FatalIfErrorf(apiextensionsv1.AddToScheme(scheme), "Cannot add api-extensions APIs to scheme")

	var clientOpts client.Options
	if !cli.Start.EnableSecretCache {
		clientOpts.Cache = &client.CacheOptions{DisableFor: []client.Object{&corev1.Secret{}}}
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:           scheme,
		Client:           clientOpts,
		LeaderElection:   cli.Start.LeaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-fusionauth",
		Cache: cache.Options{
			SyncPeriod:       &cli.Start.SyncPeriod,
			DefaultTransform: cache.TransformStripManagedFields(),
			// A ByObject transform replaces DefaultTransform for that type;
			// TransformStripCRDSchema strips managed fields as well.
			ByObject: map[client.Object]cache.ByObject{
				&apiextensionsv1.CustomResourceDefinition{}: {
					Transform: customresourcesgate.TransformStripCRDSchema,
				},
			},
		},
		Metrics: metricsserver.Options{
			BindAddress: cli.Start.MetricsBindAddress,
		},
		WebhookServer: webhook.NewServer(
			webhook.Options{
				CertDir: string(cli.Start.CertsDir),
				Port:    cli.Start.WebhookPort,
			}),
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	ctx.FatalIfErrorf(err, "Cannot create controller manager")

	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()

	metrics.Registry.MustRegister(metricRecorder)
	metrics.Registry.MustRegister(stateMetrics)

	clusterProvider := config.GetProvider()
	namespacedProvider := config.GetProviderNamespaced()
	o := tjcontroller.NewOperationStore(log)

	clusterOpts := tjcontroller.Options{
		Logger:                  log,
		GlobalRateLimiter:       ratelimiter.NewGlobal(cli.Start.MaxReconcileRate),
		PollInterval:            cli.Start.PollInterval,
		MaxConcurrentReconciles: cli.Start.MaxReconcileRate,
		Features:                &feature.Flags{},
		MetricOptions: &xpcontroller.MetricOptions{
			PollStateMetricInterval: cli.Start.PollStateMetricInterval,
			MRMetrics:               metricRecorder,
			MRStateMetrics:          stateMetrics,
		},
		Provider:              clusterProvider,
		SetupFn:               clients.TerraformSetupBuilder(),
		OperationTrackerStore: o,
		PollJitter:            pollJitter,
		StartWebhooks:         cli.Start.CertsDir != "",
	}

	namespacedOpts := tjcontroller.Options{
		Logger:                  log,
		GlobalRateLimiter:       ratelimiter.NewGlobal(cli.Start.MaxReconcileRate),
		PollInterval:            cli.Start.PollInterval,
		MaxConcurrentReconciles: cli.Start.MaxReconcileRate,
		Features:                &feature.Flags{},
		MetricOptions: &xpcontroller.MetricOptions{
			PollStateMetricInterval: cli.Start.PollStateMetricInterval,
			MRMetrics:               metricRecorder,
			MRStateMetrics:          stateMetrics,
		},
		Provider:              namespacedProvider,
		SetupFn:               clients.TerraformSetupBuilder(),
		OperationTrackerStore: o,
		PollJitter:            pollJitter,
		StartWebhooks:         cli.Start.CertsDir != "",
	}

	if cli.Start.EnableManagementPolicies {
		clusterOpts.Features.Enable(features.EnableBetaManagementPolicies)
		namespacedOpts.Features.Enable(features.EnableBetaManagementPolicies)
		log.Info("Beta feature enabled", "flag", features.EnableBetaManagementPolicies)
	}

	if cli.Start.EnableChangeLogs {
		clusterOpts.Features.Enable(feature.EnableAlphaChangeLogs)
		namespacedOpts.Features.Enable(feature.EnableAlphaChangeLogs)
		log.Info("Alpha feature enabled", "flag", feature.EnableAlphaChangeLogs)

		conn, err := grpc.NewClient("unix://"+cli.Start.ChangelogsSocketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
		ctx.FatalIfErrorf(err, "failed to create change logs client connection at %s", cli.Start.ChangelogsSocketPath)

		clo := xpcontroller.ChangeLogOptions{
			ChangeLogger: managed.NewGRPCChangeLogger(
				changelogsv1alpha1.NewChangeLogServiceClient(conn),
				managed.WithProviderVersion(fmt.Sprintf("provider-fusionauth:%s", version.Version))),
		}
		clusterOpts.ChangeLogOptions = &clo
		namespacedOpts.ChangeLogOptions = &clo
	}

	canSafeStart, err := canWatchCRD(context.TODO(), mgr)
	ctx.FatalIfErrorf(err, "SafeStart precheck failed")
	if canSafeStart {
		crdGate := new(gate.Gate[schema.GroupVersionKind])
		clusterOpts.Gate = crdGate
		namespacedOpts.Gate = crdGate
		ctx.FatalIfErrorf(customresourcesgate.Setup(mgr, xpcontroller.Options{
			Logger:                  log,
			Gate:                    crdGate,
			MaxConcurrentReconciles: 1,
			Groups:                  providerGroups(scheme, clusterProvider.RootGroup, namespacedProvider.RootGroup),
		}), "Cannot setup CRD gate")
		log.Info("Registering gated controllers (controllers start when their CRDs become established)")
		ctx.FatalIfErrorf(controllerCluster.SetupGated(mgr, clusterOpts), "Cannot setup cluster-scoped provider-fusionauth controllers")
		ctx.FatalIfErrorf(controllerNamespaced.SetupGated(mgr, namespacedOpts), "Cannot setup namespaced provider-fusionauth controllers")
		log.Info("All gated controllers registered", "cluster-resources", len(clusterProvider.Resources), "namespaced-resources", len(namespacedProvider.Resources))
	} else {
		log.Info("Provider has missing RBAC permissions for watching CRDs, controller SafeStart capability will be disabled")
		ctx.FatalIfErrorf(controllerCluster.Setup(mgr, clusterOpts), "Cannot setup cluster-scoped provider-fusionauth controllers")
		ctx.FatalIfErrorf(controllerNamespaced.Setup(mgr, namespacedOpts), "Cannot setup namespaced provider-fusionauth controllers")
	}

	ctx.FatalIfErrorf(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}

// serveDiffServer runs the diff gRPC server on the given network and address
// until the context is cancelled.
func serveDiffServer(ctx context.Context, log logging.Logger, network, address string) error {
	s, scheme, err := newDiffServer(log)
	if err != nil {
		return err
	}
	return s.Serve(ctx, network, address, scheme)
}

// newDiffServer returns the diff gRPC server, wired with its own provider
// configurations to compute plans offline, and the runtime scheme it decodes
// plan requests with.
func newDiffServer(log logging.Logger) (*diffserver.Server, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, nil, fmt.Errorf("cannot add Kubernetes core APIs to the diff server scheme: %w", err)
	}
	if err := apisCluster.AddToScheme(scheme); err != nil {
		return nil, nil, fmt.Errorf("cannot add cluster-scoped FusionAuth APIs to the diff server scheme: %w", err)
	}
	if err := apisNamespaced.AddToScheme(scheme); err != nil {
		return nil, nil, fmt.Errorf("cannot add namespaced FusionAuth APIs to the diff server scheme: %w", err)
	}
	s := diffserver.NewServer(
		diffserver.WithProviderConfigurations(config.GetProvider(), config.GetProviderNamespaced()),
		diffserver.WithLogger(log),
		diffserver.WithTerraformSetupFn(clients.OfflineTerraformSetupBuilder()),
	)
	return s, scheme, nil
}

// providerGroups returns every API group in the scheme that is one of the given
// root groups or a sub-group of one, so the CRD gate ignores unrelated CRDs.
func providerGroups(s *runtime.Scheme, roots ...string) []string {
	set := map[string]struct{}{}
	for gvk := range s.AllKnownTypes() {
		for _, root := range roots {
			if gvk.Group == root || strings.HasSuffix(gvk.Group, "."+root) {
				set[gvk.Group] = struct{}{}
			}
		}
	}
	return slices.Sorted(maps.Keys(set))
}

func canWatchCRD(ctx context.Context, mgr manager.Manager) (bool, error) {
	if err := authv1.AddToScheme(mgr.GetScheme()); err != nil {
		return false, err
	}
	verbs := []string{"get", "list", "watch"}
	for _, verb := range verbs {
		sar := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Group:    "apiextensions.k8s.io",
					Resource: "customresourcedefinitions",
					Verb:     verb,
				},
			},
		}
		if err := mgr.GetClient().Create(ctx, sar); err != nil {
			return false, errors.Wrapf(err, "unable to perform RBAC check for verb %s on CustomResourceDefinitions", verb)
		}
		if !sar.Status.Allowed {
			return false, nil
		}
	}
	return true, nil
}
