/*
Copyright 2026 Derek Gibbons.

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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	cronv3 "github.com/robfig/cron/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	powercapv1 "github.com/davidgibbons/k8s-powercap/api/v1"
)

const (
	EnvironmentKeyCurrentSchedule = "CURRENT_SCHEDULE"
	EnvironmentKeySuspend         = "SUSPEND"
	EnvironmentKeyScheduleJSON    = "SCHEDULES_JSON"
	EnvironmentKeyAgentImage      = "AGENT_IMAGE"
	AgentImage                    = "ghcr.io/davidgibbons/k8s-powercap-agent:latest"
	agentImage                    = AgentImage

	finalizerName        = "powercap.k8s.io/finalizer"
	controllerName       = "powercapschedule"
	configHashAnnotation = "powercap.k8s.io/config-hash"
)

var ErrInvalidTimezone = errors.New("invalid timezone")

func getAgentImage() string {
	if img := os.Getenv(EnvironmentKeyAgentImage); img != "" {
		return img
	}
	return AgentImage
}

type PowercapScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *PowercapScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.Log.WithName("powercapschedule-resource")

	ps := &powercapv1.PowercapSchedule{}
	if err := r.Get(ctx, req.NamespacedName, ps); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("PowercapSchedule not found, ignoring")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !ps.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, ps)
	}

	if !controllerutil.ContainsFinalizer(ps, finalizerName) {
		controllerutil.AddFinalizer(ps, finalizerName)
		if err := r.Update(ctx, ps); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if ps.Spec.Suspend {
		return r.handleSuspend(ctx, ps)
	}

	nextRunTime, err := r.getNextScheduleTime(ps)
	if err != nil {
		log.Error(err, "unable to calculate next schedule time")
		reason := powercapv1.ReasonScheduleError
		if errors.Is(err, ErrInvalidTimezone) {
			reason = powercapv1.ReasonTimezoneError
		}
		meta.SetStatusCondition(&ps.Status.Conditions, metav1.Condition{
			Type:               powercapv1.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             reason,
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
		})
		if updateErr := r.Status().Update(ctx, ps); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	if _, err := r.createOrUpdateDaemonSet(ctx, ps); err != nil {
		return ctrl.Result{}, err
	}

	now := time.Now()
	needsStatusUpdate := false
	if ps.Status.NextScheduleTime == nil || !ps.Status.NextScheduleTime.Time.Equal(nextRunTime) {
		ps.Status.NextScheduleTime = &metav1.Time{Time: nextRunTime}
		needsStatusUpdate = true
	}
	if now.After(nextRunTime) || now.Equal(nextRunTime) {
		if ps.Status.LastScheduleTime == nil || ps.Status.LastScheduleTime.Time.Before(nextRunTime) {
			ps.Status.LastScheduleTime = &metav1.Time{Time: nextRunTime}
			needsStatusUpdate = true
		}
	}
	if needsStatusUpdate {
		if err := r.Status().Update(ctx, ps); err != nil {
			return ctrl.Result{}, err
		}
	}

	requeueAfter := nextRunTime.Sub(now)
	if requeueAfter < 0 {
		requeueAfter = time.Minute
		log.Info("Requeuing after next schedule time", "duration", requeueAfter)
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

func (r *PowercapScheduleReconciler) handleDeletion(ctx context.Context, ps *powercapv1.PowercapSchedule) (ctrl.Result, error) {
	log := ctrl.Log.WithName("powercapschedule-resource")
	dsName := fmt.Sprintf("%s-daemon", ps.Name)

	ds := &appsv1.DaemonSet{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ps.Namespace, Name: dsName}, ds); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("DaemonSet not found, nothing to delete")
		} else {
			log.Error(err, "failed to get DaemonSet for deletion")
			return ctrl.Result{}, err
		}
	} else {
		if err := r.Delete(ctx, ds); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "failed to delete DaemonSet")
			return ctrl.Result{}, err
		}
		log.Info("Deleted DaemonSet", "name", dsName)
	}

	controllerutil.RemoveFinalizer(ps, finalizerName)
	if err := r.Update(ctx, ps); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PowercapScheduleReconciler) handleSuspend(ctx context.Context, ps *powercapv1.PowercapSchedule) (ctrl.Result, error) {
	log := ctrl.Log.WithName("powercapschedule-resource")
	dsName := fmt.Sprintf("%s-daemon", ps.Name)

	ds := &appsv1.DaemonSet{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ps.Namespace, Name: dsName}, ds); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("DaemonSet not found, nothing to delete for suspend", "name", dsName)
		} else {
			log.Error(err, "failed to get DaemonSet for suspend")
			return ctrl.Result{}, err
		}
	} else {
		if err := r.Delete(ctx, ds); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "failed to delete DaemonSet")
			return ctrl.Result{}, err
		}
		log.Info("Deleted DaemonSet due to suspend", "name", dsName)
	}

	meta.SetStatusCondition(&ps.Status.Conditions, metav1.Condition{
		Type:               powercapv1.ConditionReady,
		Status:             metav1.ConditionFalse,
		Reason:             powercapv1.ReasonSuspended,
		Message:            "PowercapSchedule is suspended",
		LastTransitionTime: metav1.Now(),
	})
	if err := r.Status().Update(ctx, ps); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PowercapScheduleReconciler) getNextScheduleTime(ps *powercapv1.PowercapSchedule) (time.Time, error) {
	if len(ps.Spec.Schedules) == 0 {
		return time.Time{}, fmt.Errorf("no schedules defined")
	}

	// Load the timezone from spec, defaulting to UTC if empty
	tz := ps.Spec.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: timezone %q: %v", ErrInvalidTimezone, tz, err)
	}

	nextTimes := make([]time.Time, 0, len(ps.Spec.Schedules))
	now := time.Now().In(loc)

	parser := cronv3.NewParser(cronv3.Minute | cronv3.Hour | cronv3.Dom | cronv3.Month | cronv3.Dow)
	for _, schedule := range ps.Spec.Schedules {
		sched, err := parser.Parse(schedule.Schedule)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse schedule %q: %w", schedule.Name, err)
		}

		nextTime := sched.Next(now)
		nextTimes = append(nextTimes, nextTime)
	}

	if len(nextTimes) > 0 {
		sort.Slice(nextTimes, func(i, j int) bool {
			return nextTimes[i].Before(nextTimes[j])
		})

		return nextTimes[0], nil
	}

	return time.Time{}, fmt.Errorf("no valid schedules found")
}

func (r *PowercapScheduleReconciler) createOrUpdateDaemonSet(ctx context.Context, ps *powercapv1.PowercapSchedule) (controllerutil.OperationResult, error) {
	log := ctrl.Log.WithName("powercapschedule-resource")

	schedulesJSON, err := json.Marshal(ps.Spec.Schedules)
	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("failed to marshal schedules: %w", err)
	}

	hash := sha256.Sum256(schedulesJSON)
	hashStr := hex.EncodeToString(hash[:])

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-daemon", ps.Name),
			Namespace: ps.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, ds, func() error {
		if err := controllerutil.SetControllerReference(ps, ds, r.Scheme); err != nil {
			return err
		}

		if ds.Labels == nil {
			ds.Labels = make(map[string]string)
		}
		ds.Labels["app.kubernetes.io/name"] = controllerName
		ds.Labels["app.kubernetes.io/instance"] = ps.Name

		if ds.Spec.Template.Labels == nil {
			ds.Spec.Template.Labels = make(map[string]string)
		}
		ds.Spec.Template.Labels["app.kubernetes.io/name"] = controllerName
		ds.Spec.Template.Labels["app.kubernetes.io/instance"] = ps.Name

		ds.Spec.Template.Spec.HostNetwork = false

		if ds.Spec.Template.Spec.Tolerations == nil {
			ds.Spec.Template.Spec.Tolerations = []corev1.Toleration{
				{
					Key:      "node-role.kubernetes.io/master",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
				{
					Key:      "node-role.kubernetes.io/control-plane",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}
		}

		ds.Spec.Template.Spec.NodeSelector = ps.Spec.NodeSelector

		if ds.Spec.Template.Annotations == nil {
			ds.Spec.Template.Annotations = make(map[string]string)
		}
		ds.Spec.Template.Annotations[configHashAnnotation] = hashStr

		ds.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "sys",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/sys",
						Type: func() *corev1.HostPathType { t := corev1.HostPathDirectory; return &t }(),
					},
				},
			},
		}

		image := getAgentImage()
		runAsUser := int64(0)
		runAsGroup := int64(0)
		ds.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:            "agent",
				Image:           image,
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: &corev1.SecurityContext{
					Privileged: func() *bool { b := true; return &b }(),
					RunAsUser:  &runAsUser,
					RunAsGroup: &runAsGroup,
				},
				Env: []corev1.EnvVar{
					{
						Name:  "SCHEDULES_JSON",
						Value: string(schedulesJSON),
					},
					{
						Name:  "TIMEZONE",
						Value: ps.Spec.TimeZone,
					},
					{
						Name: "NAMESPACE",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.namespace",
							},
						},
					},
					{
						Name: "POD_NAME",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.name",
							},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "sys",
						MountPath: "/sys",
						ReadOnly:  false,
					},
				},
			},
		}

		ds.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: ds.Spec.Template.Labels,
		}

		return nil
	})

	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("failed to create/update DaemonSet: %w", err)
	}

	log.Info("DaemonSet synced", "hash", hashStr)

	return op, nil
}

func (r *PowercapScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&powercapv1.PowercapSchedule{}).
		Named(controllerName).
		Owns(&appsv1.DaemonSet{}).
		Complete(r)
}
