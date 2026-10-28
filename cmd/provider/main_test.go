package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	apisCluster "github.com/lansweeper-oss/provider-fusionauth/apis/cluster"
	apisNamespaced "github.com/lansweeper-oss/provider-fusionauth/apis/namespaced"
	"github.com/lansweeper-oss/provider-fusionauth/config"
)

// Every provider group must reach the CRD gate, otherwise its controllers never start.
func TestProviderGroupsCoversAllProviderTypes(t *testing.T) {
	providerOnly := runtime.NewScheme()
	require.NoError(t, apisCluster.AddToScheme(providerOnly))
	require.NoError(t, apisNamespaced.AddToScheme(providerOnly))

	full := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(full))
	require.NoError(t, apisCluster.AddToScheme(full))
	require.NoError(t, apisNamespaced.AddToScheme(full))

	want := map[string]bool{}
	for gvk := range providerOnly.AllKnownTypes() {
		if gvk.Group != "" { // metav1 types registered in the core group, no CRD
			want[gvk.Group] = true
		}
	}

	got := providerGroups(full, config.GetProvider().RootGroup, config.GetProviderNamespaced().RootGroup)
	require.Len(t, got, len(want))
	for _, g := range got {
		require.True(t, want[g], "unexpected group %q", g)
	}
}
