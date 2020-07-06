/*
Copyright 2019 The Kubermatic Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testutil

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubermatic/utils/pkg/util"
)

type CleanUpStrategy string

const (
	CleanupAlways    CleanUpStrategy = "always"
	CleanupOnSuccess CleanUpStrategy = "on-success"
	CleanupNever     CleanUpStrategy = "never"
)

type RecordingClient struct {
	t *testing.T
	*util.ClientWatcher
	scheme          *runtime.Scheme
	objects         map[string]runtime.Object
	order           []string
	cleanUpStrategy CleanUpStrategy
	mux             sync.Mutex
}

func NewRecordingClient(t *testing.T, conf *rest.Config, scheme *runtime.Scheme, strategy CleanUpStrategy) *RecordingClient {
	log := NewLogger(t)
	cw, err := util.NewClientWatcher(conf, scheme, log)
	require.NoError(t, err)
	return &RecordingClient{
		ClientWatcher:   cw,
		scheme:          scheme,
		objects:         map[string]runtime.Object{},
		t:               t,
		cleanUpStrategy: strategy,
	}
}

var _ client.Client = (*RecordingClient)(nil)

func (rc *RecordingClient) key(obj runtime.Object) string {
	return util.ToObjectReference(obj, rc.scheme).String()
}

func (rc *RecordingClient) RegisterForCleanup(obj runtime.Object) {
	rc.mux.Lock()
	defer rc.mux.Unlock()

	key := rc.key(obj)
	rc.objects[key] = obj
	rc.order = append(rc.order, key)
}

func (rc *RecordingClient) UnregisterForCleanup(obj runtime.Object) {
	rc.mux.Lock()
	defer rc.mux.Unlock()

	key := rc.key(obj)
	delete(rc.objects, key)
}

func (rc *RecordingClient) CleanUpFunc(ctx context.Context) func() {
	return func() {
		rc.t.Helper()
		switch rc.cleanUpStrategy {
		case CleanupNever:
			return
		case CleanupOnSuccess:
			if rc.t.Failed() {
				return
			}
		case CleanupAlways:
			break
		default:
			rc.t.Logf("unknown cleanup strategy: %v", rc.cleanUpStrategy)
			rc.t.FailNow()
		}

		// cleanup in reverse order of creation
		for i := len(rc.order) - 1; i >= 0; i-- {
			key := rc.order[i]
			obj, ok := rc.objects[key]
			if !ok {
				continue
			}

			err := DeleteAndWaitUntilNotFound(ctx, rc, obj, WithTimeout(time.Minute))
			if err != nil {
				err = fmt.Errorf("cleanup %s: %w", key, err)
			}
			require.NoError(rc.t, err)
		}
	}
}

func (rc *RecordingClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	rc.t.Helper()
	rc.t.Logf("creating %s", util.MustLogLine(obj, rc.scheme))
	rc.RegisterForCleanup(obj)
	return rc.ClientWatcher.Create(ctx, obj, opts...)
}

func (rc *RecordingClient) EnsureCreated(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	rc.t.Helper()
	rc.t.Logf("creating %s", util.MustLogLine(obj, rc.scheme))
	rc.RegisterForCleanup(obj)
	oldObj := obj.DeepCopyObject()
	err := rc.ClientWatcher.Create(ctx, obj, opts...)
	if err != nil && errors.IsAlreadyExists(err) {
		rc.t.Logf("alreadyExists, update %s", util.MustLogLine(obj, rc.scheme))
		updateErr := rc.ClientWatcher.Update(ctx, oldObj)
		if err := rc.scheme.Convert(oldObj, obj, nil); err != nil {
			return err
		}
		return updateErr
	}
	return nil
}

func (rc *RecordingClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	rc.t.Helper()
	rc.t.Logf("deleting %s", util.MustLogLine(obj, rc.scheme))
	rc.UnregisterForCleanup(obj)
	return rc.ClientWatcher.Delete(ctx, obj, opts...)
}
