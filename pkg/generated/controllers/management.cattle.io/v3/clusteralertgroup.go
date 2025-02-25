/*
Copyright 2023 Rancher Labs, Inc.

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

// Code generated by main. DO NOT EDIT.

package v3

import (
	"context"
	"time"

	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ClusterAlertGroupController interface for managing ClusterAlertGroup resources.
type ClusterAlertGroupController interface {
	generic.ControllerInterface[*v3.ClusterAlertGroup, *v3.ClusterAlertGroupList]
}

// ClusterAlertGroupClient interface for managing ClusterAlertGroup resources in Kubernetes.
type ClusterAlertGroupClient interface {
	generic.ClientInterface[*v3.ClusterAlertGroup, *v3.ClusterAlertGroupList]
}

// ClusterAlertGroupCache interface for retrieving ClusterAlertGroup resources in memory.
type ClusterAlertGroupCache interface {
	generic.CacheInterface[*v3.ClusterAlertGroup]
}

type ClusterAlertGroupStatusHandler func(obj *v3.ClusterAlertGroup, status v3.AlertStatus) (v3.AlertStatus, error)

type ClusterAlertGroupGeneratingHandler func(obj *v3.ClusterAlertGroup, status v3.AlertStatus) ([]runtime.Object, v3.AlertStatus, error)

func RegisterClusterAlertGroupStatusHandler(ctx context.Context, controller ClusterAlertGroupController, condition condition.Cond, name string, handler ClusterAlertGroupStatusHandler) {
	statusHandler := &clusterAlertGroupStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, generic.FromObjectHandlerToHandler(statusHandler.sync))
}

func RegisterClusterAlertGroupGeneratingHandler(ctx context.Context, controller ClusterAlertGroupController, apply apply.Apply,
	condition condition.Cond, name string, handler ClusterAlertGroupGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &clusterAlertGroupGeneratingHandler{
		ClusterAlertGroupGeneratingHandler: handler,
		apply:                              apply,
		name:                               name,
		gvk:                                controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterClusterAlertGroupStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type clusterAlertGroupStatusHandler struct {
	client    ClusterAlertGroupClient
	condition condition.Cond
	handler   ClusterAlertGroupStatusHandler
}

func (a *clusterAlertGroupStatusHandler) sync(key string, obj *v3.ClusterAlertGroup) (*v3.ClusterAlertGroup, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type clusterAlertGroupGeneratingHandler struct {
	ClusterAlertGroupGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *clusterAlertGroupGeneratingHandler) Remove(key string, obj *v3.ClusterAlertGroup) (*v3.ClusterAlertGroup, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v3.ClusterAlertGroup{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *clusterAlertGroupGeneratingHandler) Handle(obj *v3.ClusterAlertGroup, status v3.AlertStatus) (v3.AlertStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.ClusterAlertGroupGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
