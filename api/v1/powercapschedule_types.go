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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionReady  = "Ready"
	ConditionActive = "Active"

	ReasonTimezoneError = "TimezoneError"
	ReasonScheduleError = "ScheduleError"
	ReasonSuspended     = "Suspended"
)

// PowercapScheduleSpec defines the desired state of PowercapSchedule
type PowercapScheduleSpec struct {
	// NodeSelector is a selector which must be true for the DaemonSet to match a node.
	// Pods are only scheduled on nodes matching these labels.
	// +optional
	// +kubebuilder:default={}
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// TimeZone is the timezone for the schedule. If not specified, defaults to UTC.
	// Example: "America/Los_Angeles", "UTC", "Europe/Paris"
	// +optional
	// +kubebuilder:default=UTC
	TimeZone string `json:"timeZone,omitempty"`

	// Suspend suspends the scheduling of power limit changes. When true, no new power limits are applied.
	// +optional
	// +kubebuilder:default=false
	Suspend bool `json:"suspend,omitempty"`

	// Schedules is a list of power management rules.
	// Each schedule can define when to apply power limits and to which zones/constraints.
	// +kubebuilder:validation:Required
	Schedules []PowercapRule `json:"schedules"`
}

// PowercapRule defines a single power management schedule.
type PowercapRule struct {
	// Name is a unique identifier for this schedule (e.g., "workday", "night", "backup").
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Schedule is a cron expression defining when to apply the power limits.
	// Minute and hour must be specific values; day/month/dow may use ranges.
	// Format: "Minutes Hours Day-of-Month Month Day-of-Week"
	// Examples: "0 9 * * 1" (9 AM Monday), "0 9 * * 1-5" (9 AM Mon-Fri)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^(\\d+|\\*) (\\d+|\\*) (\\d+(-\\d+)?|\\*) (\\d+(-\\d+)?|\\*) (\\d+(-\\d+)?|\\*)$"
	Schedule string `json:"schedule"`

	// PowerLimits defines the power limits to apply for this schedule.
	// Each entry can target a specific zone/constraint combination.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	PowerLimits []PowerLimitEntry `json:"powerLimits"`
}

// PowerLimitEntry defines a power limit for a specific zone and constraint.
type PowerLimitEntry struct {
	// Zone specifies the powercap zone to control.
	// Example: "intel-rapl:0", "intel-rapl:1"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9:_-]+$"
	Zone string `json:"zone"`

	// Constraint specifies which constraint within the powercap zone to target.
	// Example: "constraint_0", "constraint_1"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^constraint_[0-9]+$"
	Constraint string `json:"constraint"`

	// PowerLimit is the power limit to apply, in microwatts (µW).
	// Example: 65000000 = 65 Watts
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000000000
	PowerLimit int64 `json:"powerLimit"`
}

// PowercapScheduleStatus defines the observed state of PowercapSchedule.
type PowercapScheduleStatus struct {
	// LastScheduleTime is the last time any schedule was triggered.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// NextScheduleTime is the next time the earliest schedule is due to trigger.
	// +optional
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`

	// ScheduleStatuses tracks the status of each individual schedule.
	// +optional
	ScheduleStatuses []ScheduleStatus `json:"scheduleStatuses,omitempty"`

	// conditions represent the current state of the PowercapSchedule resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ScheduleStatus tracks the state of a single schedule.
type ScheduleStatus struct {
	// Name is the name of the schedule from the spec.
	Name string `json:"name"`

	// LastScheduleTime is the last time this schedule was triggered.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// NextScheduleTime is the next time this schedule is due to trigger.
	// +optional
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`

	// CurrentPowerLimit is the current power limit applied by this schedule, in microwatts.
	// +optional
	CurrentPowerLimit int64 `json:"currentPowerLimit,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:annotations="api-approved.kubernetes.io=unapproved, experimental-only"
// +kubebuilder:webhook:path=/mutate-powercap-k8s-io-v1-powercapschedule,mutating=true,failurePolicy=fail,sideEffects=None,groups=powercap.k8s.io,resources=powercapschedules,verbs=create;update,versions=v1,name=mpowercapschedule.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-powercap-k8s-io-v1-powercapschedule,mutating=false,failurePolicy=fail,sideEffects=None,groups=powercap.k8s.io,resources=powercapschedules,verbs=create;update,versions=v1,name=vpowercapschedule.kb.io,admissionReviewVersions=v1

// PowercapSchedule is the Schema for the powercapschedules API
type PowercapSchedule struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of PowercapSchedule
	// +required
	Spec PowercapScheduleSpec `json:"spec"`

	// status defines the observed state of PowercapSchedule
	// +optional
	Status PowercapScheduleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PowercapScheduleList contains a list of PowercapSchedule
type PowercapScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PowercapSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PowercapSchedule{}, &PowercapScheduleList{})
}
