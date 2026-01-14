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

package v1

import (
	"context"
	"fmt"
	"regexp"
	"time"

	cronv3 "github.com/robfig/cron/v3"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	powercapv1 "github.com/davidgibbons/k8s-powercap/api/v1"
)

var powercapScheduleLog = ctrl.Log.WithName("powercapschedule-resource")

// SetupPowercapScheduleWebhookWithManager registers webhook for PowercapSchedule in manager.
func SetupPowercapScheduleWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&powercapv1.PowercapSchedule{}).
		WithValidator(&PowercapScheduleCustomValidator{}).
		WithDefaulter(&PowercapScheduleCustomDefaulter{}).
		Complete()
}

// PowercapScheduleCustomDefaulter struct is responsible for setting default values on custom resource of
// Kind PowercapSchedule when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type PowercapScheduleCustomDefaulter struct {
}

var _ webhook.CustomDefaulter = &PowercapScheduleCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for Kind PowercapSchedule.
func (d *PowercapScheduleCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	powercapschedule, ok := obj.(*powercapv1.PowercapSchedule)

	if !ok {
		return fmt.Errorf("expected a PowercapSchedule object but got %T", obj)
	}

	powercapScheduleLog.Info("Defaulting for PowercapSchedule", "name", powercapschedule.GetName())

	// Set default timezone to UTC if not specified
	if powercapschedule.Spec.TimeZone == "" {
		powercapschedule.Spec.TimeZone = "UTC"
	}

	// Validate and default empty schedule names
	for _, schedule := range powercapschedule.Spec.Schedules {
		if schedule.Name == "" {
			schedule.Name = fmt.Sprintf("schedule-%d", time.Now().Unix())
		}
	}

	return nil
}

// PowercapScheduleCustomValidator struct is responsible for validating PowercapSchedule resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type PowercapScheduleCustomValidator struct {
}

var _ webhook.CustomValidator = &PowercapScheduleCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for type PowercapSchedule.
func (v *PowercapScheduleCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	powercapschedule, ok := obj.(*powercapv1.PowercapSchedule)
	if !ok {
		return nil, fmt.Errorf("expected a PowercapSchedule object but got %T", obj)
	}

	powercapScheduleLog.Info("Validation for PowercapSchedule upon creation", "name", powercapschedule.GetName())

	return v.validateSchedules(powercapschedule.Spec.Schedules)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for type PowercapSchedule.
func (v *PowercapScheduleCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	powercapschedule, ok := newObj.(*powercapv1.PowercapSchedule)
	if !ok {
		return nil, fmt.Errorf("expected a PowercapSchedule object for newObj but got %T", newObj)
	}

	powercapScheduleLog.Info("Validation for PowercapSchedule upon update", "name", powercapschedule.GetName())

	return v.validateSchedules(powercapschedule.Spec.Schedules)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for type PowercapSchedule.
func (v *PowercapScheduleCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	powercapschedule, ok := obj.(*powercapv1.PowercapSchedule)
	if !ok {
		return nil, fmt.Errorf("expected a PowercapSchedule object but got %T", obj)
	}

	powercapScheduleLog.Info("Validation for PowercapSchedule upon deletion", "name", powercapschedule.GetName())

	return nil, nil
}

// singlePointCronPattern validates cron expressions: 5 fields, each a number or wildcard.
var singlePointCronPattern = regexp.MustCompile(`^((\d+|\*) ){4}(\d+|\*)$`)

// validateSchedules validates all schedules in the CRD.
func (v *PowercapScheduleCustomValidator) validateSchedules(schedules []powercapv1.PowercapRule) (admission.Warnings, error) {
	if len(schedules) == 0 {
		return admission.Warnings{}, fmt.Errorf("at least one schedule must be defined")
	}

	scheduleNames := make(map[string]bool)
	for _, schedule := range schedules {
		if schedule.Name == "" {
			return nil, fmt.Errorf("schedule name is required")
		}

		if scheduleNames[schedule.Name] {
			return nil, fmt.Errorf("duplicate schedule name %q", schedule.Name)
		}
		scheduleNames[schedule.Name] = true
	}

	for _, schedule := range schedules {
		parser := cronv3.NewParser(cronv3.Minute | cronv3.Hour | cronv3.Dom | cronv3.Month | cronv3.Dow)
		if _, err := parser.Parse(schedule.Schedule); err != nil {
			return nil, fmt.Errorf("invalid cron schedule %q in schedule %q: %w", schedule.Schedule, schedule.Name, err)
		}

		if !singlePointCronPattern.MatchString(schedule.Schedule) {
			return nil, fmt.Errorf("schedule %q in schedule %q must represent a single point in time (no comma lists or step values)", schedule.Schedule, schedule.Name)
		}

		if len(schedule.PowerLimits) == 0 {
			return nil, fmt.Errorf("schedule %q must have at least one power limit entry", schedule.Name)
		}

		for _, limit := range schedule.PowerLimits {
			if limit.PowerLimit <= 0 {
				return nil, fmt.Errorf("power limit must be greater than 0, got %d in schedule %q", limit.PowerLimit, schedule.Name)
			}

			if limit.PowerLimit > 1000000000 {
				return nil, fmt.Errorf("power limit %d in schedule %q exceeds maximum of 1000 watts", limit.PowerLimit, schedule.Name)
			}

			if limit.Zone == "" {
				return nil, fmt.Errorf("zone must be specified in power limit entry")
			}

			if limit.Constraint == "" {
				return nil, fmt.Errorf("constraint must be specified in power limit entry")
			}
		}
	}

	return nil, nil
}
