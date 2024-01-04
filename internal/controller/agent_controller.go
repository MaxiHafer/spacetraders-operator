/*
Copyright 2024.

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

package controller

import (
	"context"
	"errors"
	spacetradersv1alpha1 "github.com/maxihafer/spacetraders-operator/api/v1alpha1"
	"github.com/maxihafer/spacetraders-operator/pkg/spacetraders"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	typeRegisteredAgent = "Registered"
)

// AgentReconcilerConfig is the configuration for the AgentReconciler
type AgentReconcilerConfig struct {
	EMail string `envconfig:"ACCOUNT_EMAIL" required:"true"`
}

// AgentReconciler reconciles a Agent object
type AgentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	Spacetraders spacetraders.ClientWithResponsesInterface
	Config       *AgentReconcilerConfig
}

func (r *AgentReconciler) assertAgentPresent(ctx context.Context, body spacetraders.RegisterJSONRequestBody) (*string, error) {
	registerResponse, err := r.Spacetraders.RegisterWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}

	if registerResponse.StatusCode() != http.StatusCreated {
		return nil, spacetraders.NewAPIError(registerResponse.StatusCode(), registerResponse.Body)
	}

	return &registerResponse.JSON201.Data.Token, nil
}

//+kubebuilder:rbac:groups=spacetraders.hafer.dev,resources=agents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=spacetraders.hafer.dev,resources=agents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	agent := &spacetradersv1alpha1.Agent{}
	err := r.Get(ctx, req.NamespacedName, agent)
	if err != nil {
		logger.Info("Agent resource not found. Ignoring since object must be deleted")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if agent.Status.Conditions == nil || len(agent.Status.Conditions) == 0 {
		meta.SetStatusCondition(&agent.Status.Conditions, metav1.Condition{
			Type:    typeRegisteredAgent,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting to reconcile Agent",
		})
		if err = r.Status().Update(ctx, agent); err != nil {
			logger.Error(err, "Failed to update Agent status")
			return ctrl.Result{}, err
		}

		if err := r.Get(ctx, req.NamespacedName, agent); err != nil {
			logger.Error(err, "Failed to re-fetch Agent")
			return ctrl.Result{}, err
		}
	}

	found := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: agent.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		token, err := r.assertAgentPresent(ctx, spacetraders.RegisterJSONRequestBody{
			Symbol:  agent.Spec.Symbol,
			Faction: spacetraders.FactionSymbol(agent.Spec.Faction),
			Email:   &r.Config.EMail,
		})
		if err != nil {
			apiError := &spacetraders.APIError{}
			if errors.As(err, apiError) {
				meta.SetStatusCondition(&agent.Status.Conditions, metav1.Condition{
					Type:    typeRegisteredAgent,
					Status:  metav1.ConditionFalse,
					Reason:  "Failed",
					Message: apiError.Message,
				})
				if err := r.Status().Update(ctx, agent); err != nil {
					logger.Error(err, "Failed to update Agent status")
					return ctrl.Result{}, err
				}

				logger.Error(err, "Failed to register agent")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&agent.Status.Conditions, metav1.Condition{
				Type:    typeRegisteredAgent,
				Status:  metav1.ConditionFalse,
				Reason:  "Error",
				Message: err.Error(),
			})

			if err := r.Status().Update(ctx, agent); err != nil {
				logger.Error(err, "Error while updating Agent status")
				return ctrl.Result{}, err
			}

			logger.Error(err, "Failed to register agent")
			return ctrl.Result{}, err
		}

		// Define a new Secret object
		accessTokenSecret, err := r.accessTokenSecretForAgent(agent, *token)
		if err != nil {
			meta.SetStatusCondition(&agent.Status.Conditions, metav1.Condition{
				Type:    typeRegisteredAgent,
				Status:  metav1.ConditionFalse,
				Reason:  "Error",
				Message: err.Error(),
			})

			if err := r.Status().Update(ctx, agent); err != nil {
				logger.Error(err, "Error while updating Agent status")
				return ctrl.Result{}, err
			}

			logger.Error(err, "Failed to create new Secret")
			return ctrl.Result{}, err
		}

		logger.Info("Creating a new Secret", "Secret.Namespace", accessTokenSecret.Namespace, "Secret.Name", accessTokenSecret.Name)
		err = r.Create(ctx, accessTokenSecret)
		if err != nil {
			logger.Error(err, "Failed to create new Secret", "Secret.Namespace", accessTokenSecret.Namespace, "Secret.Name", accessTokenSecret.Name)
			return ctrl.Result{}, err
		}

		// Secret created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	if err != nil {
		meta.SetStatusCondition(&agent.Status.Conditions, metav1.Condition{
			Type:    typeRegisteredAgent,
			Status:  metav1.ConditionFalse,
			Reason:  "Error",
			Message: err.Error(),
		})
		if err := r.Status().Update(ctx, agent); err != nil {
			logger.Error(err, "Failed to update Agent status")
			return ctrl.Result{Requeue: true}, err
		}

		logger.Error(err, "Failed to register agent")
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *AgentReconciler) accessTokenSecretForAgent(agent *spacetradersv1alpha1.Agent, token string) (*corev1.Secret, error) {
	ls := labelsForAccessTokenSecret(agent.Name)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels:    ls,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"access-token": token,
		},
		Immutable: pointer.Bool(true),
	}

	if err := ctrl.SetControllerReference(agent, secret, r.Scheme); err != nil {
		return nil, err
	}
	return secret, nil
}

func labelsForAccessTokenSecret(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "spacetraders-operator",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/part-of":    "spacetraders-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&spacetradersv1alpha1.Agent{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
