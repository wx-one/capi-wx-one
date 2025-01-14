/*
Copyright 2025.

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

package v1beta1

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	infrastructurev1beta1 "github.com/wx-one/cluster-api-provider-wxone/api/v1beta1"
)

// nolint:unused
// log is for logging in this package.
var wxoneclustertemplatelog = logf.Log.WithName("wxoneclustertemplate-resource")

// SetupWXOneClusterTemplateWebhookWithManager registers the webhook for WXOneClusterTemplate in the manager.
func SetupWXOneClusterTemplateWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&infrastructurev1beta1.WXOneClusterTemplate{}).
		WithValidator(&WXOneClusterTemplateCustomValidator{}).
		WithDefaulter(&WXOneClusterTemplateCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-wxoneclustertemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=wxoneclustertemplates,verbs=create;update,versions=v1beta1,name=mwxoneclustertemplate-v1beta1.kb.io,admissionReviewVersions=v1

// WXOneClusterTemplateCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind WXOneClusterTemplate when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type WXOneClusterTemplateCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &WXOneClusterTemplateCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind WXOneClusterTemplate.
func (d *WXOneClusterTemplateCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	wxoneclustertemplate, ok := obj.(*infrastructurev1beta1.WXOneClusterTemplate)

	if !ok {
		return fmt.Errorf("expected an WXOneClusterTemplate object but got %T", obj)
	}
	wxoneclustertemplatelog.Info("Defaulting for WXOneClusterTemplate", "name", wxoneclustertemplate.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-wxoneclustertemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=wxoneclustertemplates,verbs=create;update,versions=v1beta1,name=vwxoneclustertemplate-v1beta1.kb.io,admissionReviewVersions=v1

// WXOneClusterTemplateCustomValidator struct is responsible for validating the WXOneClusterTemplate resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type WXOneClusterTemplateCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &WXOneClusterTemplateCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type WXOneClusterTemplate.
func (v *WXOneClusterTemplateCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	wxoneclustertemplate, ok := obj.(*infrastructurev1beta1.WXOneClusterTemplate)
	if !ok {
		return nil, fmt.Errorf("expected a WXOneClusterTemplate object but got %T", obj)
	}
	wxoneclustertemplatelog.Info("Validation for WXOneClusterTemplate upon creation", "name", wxoneclustertemplate.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type WXOneClusterTemplate.
func (v *WXOneClusterTemplateCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	wxoneclustertemplate, ok := newObj.(*infrastructurev1beta1.WXOneClusterTemplate)
	if !ok {
		return nil, fmt.Errorf("expected a WXOneClusterTemplate object for the newObj but got %T", newObj)
	}
	wxoneclustertemplatelog.Info("Validation for WXOneClusterTemplate upon update", "name", wxoneclustertemplate.GetName())

	oldClusterTemplate, ok := oldObj.(*infrastructurev1beta1.WXOneClusterTemplate)
	if !ok {
		return nil, fmt.Errorf("expected a WXOneClusterTemplate object for the oldObj but got %T", newObj)
	}

	newClusterTemplateCopy := wxoneclustertemplate.DeepCopy()
	oldClusterTemplateCopy := oldClusterTemplate.DeepCopy()

	newClusterTemplateCopy.Spec.Template.Spec.ControlPlaneEndpoint = oldClusterTemplateCopy.Spec.Template.Spec.ControlPlaneEndpoint

	if !reflect.DeepEqual(newClusterTemplateCopy.Spec, oldClusterTemplateCopy.Spec) {
		return nil, fmt.Errorf("modifications to spec are not allowed")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type WXOneClusterTemplate.
func (v *WXOneClusterTemplateCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	wxoneclustertemplate, ok := obj.(*infrastructurev1beta1.WXOneClusterTemplate)
	if !ok {
		return nil, fmt.Errorf("expected a WXOneClusterTemplate object but got %T", obj)
	}
	wxoneclustertemplatelog.Info("Validation for WXOneClusterTemplate upon deletion", "name", wxoneclustertemplate.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
