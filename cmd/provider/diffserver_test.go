package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	diffv1alpha1 "github.com/crossplane/upjet/v2/proto/diff/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// The manifests below reference ProviderConfigs and Secrets that exist
// nowhere: the offline diff server needs neither credentials nor FusionAuth.
const (
	groupActual = `
apiVersion: fusionauth.crossplane.io/v1alpha1
kind: Group
metadata:
  name: admins
  annotations:
    crossplane.io/external-name: 0b5b1b8e-6a5b-4b8e-9a7e-1f2d3c4b5a69
spec:
  providerConfigRef:
    name: missing
  forProvider:
    name: Admins
    tenantId: 7f9c2b1a-3d4e-4f5a-8b6c-9d0e1f2a3b4c
status:
  atProvider:
    id: 0b5b1b8e-6a5b-4b8e-9a7e-1f2d3c4b5a69
    groupId: 0b5b1b8e-6a5b-4b8e-9a7e-1f2d3c4b5a69
    name: Admins
    tenantId: 7f9c2b1a-3d4e-4f5a-8b6c-9d0e1f2a3b4c
`
	groupRenamed = `
apiVersion: fusionauth.crossplane.io/v1alpha1
kind: Group
metadata:
  name: admins
  annotations:
    crossplane.io/external-name: 0b5b1b8e-6a5b-4b8e-9a7e-1f2d3c4b5a69
spec:
  providerConfigRef:
    name: missing
  forProvider:
    name: Administrators
    tenantId: 7f9c2b1a-3d4e-4f5a-8b6c-9d0e1f2a3b4c
`
	groupNewID = `
apiVersion: fusionauth.crossplane.io/v1alpha1
kind: Group
metadata:
  name: admins
  annotations:
    crossplane.io/external-name: 0b5b1b8e-6a5b-4b8e-9a7e-1f2d3c4b5a69
spec:
  providerConfigRef:
    name: missing
  forProvider:
    groupId: 5e6f7a8b-9c0d-4e1f-a2b3-c4d5e6f7a8b9
    name: Admins
    tenantId: 7f9c2b1a-3d4e-4f5a-8b6c-9d0e1f2a3b4c
`
	// keyWoVersion is only replaced through upstream's CustomizeDiff, the one
	// place the plan hands the Terraform provider its meta.
	apiKeyActual = `
apiVersion: fusionauth.crossplane.io/v1alpha1
kind: APIKey
metadata:
  name: ci
  annotations:
    crossplane.io/external-name: 2c3d4e5f-6a7b-4c8d-9e0f-1a2b3c4d5e6f
spec:
  providerConfigRef:
    name: missing
  forProvider:
    description: ci
    keyWoVersion: 1
status:
  atProvider:
    id: 2c3d4e5f-6a7b-4c8d-9e0f-1a2b3c4d5e6f
    description: ci
    keyWoVersion: 1
`
	apiKeyRotated = `
apiVersion: fusionauth.crossplane.io/v1alpha1
kind: APIKey
metadata:
  name: ci
  annotations:
    crossplane.io/external-name: 2c3d4e5f-6a7b-4c8d-9e0f-1a2b3c4d5e6f
spec:
  providerConfigRef:
    name: missing
  forProvider:
    description: ci
    keyWoVersion: 2
`
	namespacedGroup = `
apiVersion: fusionauth.m.crossplane.io/v1alpha1
kind: Group
metadata:
  name: readers
  namespace: default
spec:
  providerConfigRef:
    kind: ClusterProviderConfig
    name: missing
  forProvider:
    name: Readers
    tenantId: 7f9c2b1a-3d4e-4f5a-8b6c-9d0e1f2a3b4c
`
)

// TestDiffServerPlan drives the diff gRPC server the way a client does, over a
// unix socket, wired exactly as main wires it.
func TestDiffServerPlan(t *testing.T) {
	ctx := context.Background()
	s, scheme, err := newDiffServer(logging.NewNopLogger())
	require.NoError(t, err)

	// A short path: unix socket paths are limited to ~104 bytes on macOS.
	dir, err := os.MkdirTemp("", "diff")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "diff.sock")

	serveCtx, cancel := context.WithCancel(ctx)
	served := make(chan error, 1)
	go func() { served <- s.Serve(serveCtx, "unix", sock, scheme) }()
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-served)
	})

	conn, err := grpc.NewClient("unix://"+sock, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := diffv1alpha1.NewPlanServiceClient(conn)

	cases := map[string]struct {
		desired, actual string
		action          diffv1alpha1.Action
		changes         map[string]string
	}{
		"ClusterGroupCreate": {
			desired: groupActual,
			action:  diffv1alpha1.Action_ACTION_CREATE,
			changes: map[string]string{
				"spec.forProvider.name":     "<unset> -> Admins",
				"spec.forProvider.tenantId": "<unset> -> 7f9c2b1a-3d4e-4f5a-8b6c-9d0e1f2a3b4c",
			},
		},
		"ClusterGroupNoOp": {
			desired: groupActual,
			actual:  groupActual,
			action:  diffv1alpha1.Action_ACTION_NO_OP,
		},
		"ClusterGroupRenameUpdates": {
			desired: groupRenamed,
			actual:  groupActual,
			action:  diffv1alpha1.Action_ACTION_UPDATE,
			changes: map[string]string{
				"spec.forProvider.name": "Admins -> Administrators",
			},
		},
		"ClusterGroupNewIDReplaces": {
			desired: groupNewID,
			actual:  groupActual,
			action:  diffv1alpha1.Action_ACTION_REPLACE,
			changes: map[string]string{
				"spec.forProvider.groupId": "0b5b1b8e-6a5b-4b8e-9a7e-1f2d3c4b5a69 -> 5e6f7a8b-9c0d-4e1f-a2b3-c4d5e6f7a8b9 (requires replace)",
			},
		},
		"ClusterAPIKeyRotateReplaces": {
			desired: apiKeyRotated,
			actual:  apiKeyActual,
			action:  diffv1alpha1.Action_ACTION_REPLACE,
			changes: map[string]string{
				"spec.forProvider.keyWoVersion": "1 -> 2 (requires replace)",
			},
		},
		"NamespacedGroupCreate": {
			desired: namespacedGroup,
			action:  diffv1alpha1.Action_ACTION_CREATE,
			changes: map[string]string{
				"spec.forProvider.name":     "<unset> -> Readers",
				"spec.forProvider.tenantId": "<unset> -> 7f9c2b1a-3d4e-4f5a-8b6c-9d0e1f2a3b4c",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			callCtx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()
			rsp, err := client.Plan(callCtx, &diffv1alpha1.PlanRequest{
				DesiredResource: asStruct(t, tc.desired),
				ActualResource:  asStruct(t, tc.actual),
			}, grpc.WaitForReady(true))
			require.NoError(t, err)
			assert.Equal(t, tc.action.String(), rsp.GetAction().String())
			assert.Equal(t, tc.changes, declaredChanges(rsp))
		})
	}
}

// declaredChanges renders the changes to fields the desired resource declares
// as "actual -> planned", leaving out the Terraform provider's own defaults.
func declaredChanges(rsp *diffv1alpha1.PlanResponse) map[string]string {
	var changes map[string]string
	for _, c := range rsp.GetChanges() {
		if c.GetOrigin() != diffv1alpha1.Origin_ORIGIN_DESIRED_STATE {
			continue
		}
		change := renderValue(c.GetActual()) + " -> " + renderValue(c.GetPlanned())
		if c.GetRequiresReplace() {
			change += " (requires replace)"
		}
		if changes == nil {
			changes = map[string]string{}
		}
		changes[c.GetField()] = change
	}
	return changes
}

func renderValue(v *diffv1alpha1.FieldValue) string {
	switch {
	case v == nil:
		return "<unset>"
	case v.GetValue() != nil:
		return fmt.Sprintf("%v", v.GetValue().AsInterface())
	default:
		return v.GetAbsence().String()
	}
}

// asStruct converts a YAML manifest into a plan request's protobuf Struct. An
// empty manifest is nil: a resource that does not exist yet.
func asStruct(t *testing.T, manifest string) *structpb.Struct {
	t.Helper()
	if manifest == "" {
		return nil
	}
	j, err := yaml.ToJSON([]byte(manifest))
	require.NoError(t, err)
	s := &structpb.Struct{}
	require.NoError(t, s.UnmarshalJSON(j))
	return s
}
