//
// Copyright 2022 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kcptoolsv1alpha1 "github.com/IBM/policy-control-operator/api/v1alpha1"
)

func (r *PolicyControlReconciler) createOrUpdateTypedResource(
	ctx context.Context,
	logger logr.Logger,
	pc kcptoolsv1alpha1.PolicyControl,
	typedObj client.Object,
	isSetControllerReference bool,
) (ctrl.Result, error) {

	gvk := typedObj.GetObjectKind().GroupVersionKind().String()
	var err error
	if isSetControllerReference {
		err = controllerutil.SetControllerReference(&pc, typedObj, r.Scheme)
		if err != nil {
			logger.Error(err, "failed to set ownerReference")
			return ctrl.Result{}, err
		}
	}

	err = r.Get(ctx, client.ObjectKey{Namespace: typedObj.GetNamespace(), Name: typedObj.GetName()}, typedObj)
	if err != nil {
		err = r.Create(ctx, typedObj)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to create %s", gvk))
			return ctrl.Result{}, err
		}
	} else if err == nil {
		err = r.Update(ctx, typedObj)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to update %s", gvk))
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, err
}

func (r *PolicyControlReconciler) createOrUpdateTypedResourceByUnstructured(
	ctx context.Context,
	logger logr.Logger,
	pc kcptoolsv1alpha1.PolicyControl,
	obj unstructured.Unstructured,
	typedObj client.Object,
	isSetControllerReference bool,
) (ctrl.Result, error) {

	gvk := typedObj.GetObjectKind().GroupVersionKind().String()
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to convert resource to %s", gvk))
		return ctrl.Result{}, err
	}

	return r.createOrUpdateTypedResource(ctx, logger, pc, typedObj, isSetControllerReference)
}

func (r *PolicyControlReconciler) createOrUpdateUnstructuredResource(
	ctx context.Context,
	logger logr.Logger,
	dyClient dynamic.Interface,
	restMapping meta.RESTMapping,
	obj unstructured.Unstructured,
	ignoreUpdateError bool,
) (ctrl.Result, error) {
	_, err := dyClient.Resource(restMapping.Resource).Namespace(obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		_, err = dyClient.Resource(restMapping.Resource).Namespace(obj.GetNamespace()).Create(ctx, &obj, metav1.CreateOptions{})
		if err != nil {
			logger.Error(err, "failed to create Resource")
			return ctrl.Result{}, err
		}
	} else if err == nil {
		_, err = dyClient.Resource(restMapping.Resource).Namespace(obj.GetNamespace()).Update(ctx, &obj, metav1.UpdateOptions{})
		if err != nil {
			logger.Error(err, "failed to update Resource")
			if !ignoreUpdateError {
				return ctrl.Result{}, err
			}
		}
	} else {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *PolicyControlReconciler) createOrUpdateUnstructuredResourceFromFile(
	ctx context.Context,
	logger logr.Logger,
	restMapper meta.RESTMapper,
	dyClient dynamic.Interface,
	path string,
	ignoreUpdateError bool,
) error {
	obj, mapping, err := getUnstructuredFromFile(logger, path, restMapper)
	if err != nil {
		return err
	}
	_, err = r.createOrUpdateUnstructuredResource(ctx, logger, dyClient, *mapping, obj, ignoreUpdateError)
	if err != nil {
		return err
	}
	return nil
}

func (r *PolicyControlReconciler) createOrUpdate(
	ctx context.Context,
	logger logr.Logger,
	obj client.Object,
) error {
	err := r.Get(ctx, client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
	if err != nil {
		err = r.Create(ctx, obj)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to create %s", obj))
			return err
		}
	} else if err == nil {
		err = r.Update(ctx, obj)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to update %s", obj))
			return err
		}
	}
	return nil
}
