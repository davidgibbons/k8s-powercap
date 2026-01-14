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
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	powercapv1 "github.com/davidgibbons/k8s-powercap/api/v1"
)

var _ = Describe("PowercapSchedule Controller", func() {
	var reconciler *PowercapScheduleReconciler

	BeforeEach(func() {
		reconciler = &PowercapScheduleReconciler{}
	})

	Describe("getNextScheduleTime", func() {
		Context("with valid timezone", func() {
			It("should calculate next time in America/Los_Angeles timezone", func() {
				ps := &powercapv1.PowercapSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-tz",
						Namespace: "default",
					},
					Spec: powercapv1.PowercapScheduleSpec{
						TimeZone: "America/Los_Angeles",
						Schedules: []powercapv1.PowercapRule{
							{
								Name:     "test-schedule",
								Schedule: "0 9 * * *",
								PowerLimits: []powercapv1.PowerLimitEntry{
									{
										Zone:       "intel-rapl:0",
										Constraint: "constraint_0",
										PowerLimit: 65000000,
									},
								},
							},
						},
					},
				}

				nextTime, err := reconciler.getNextScheduleTime(ps)
				Expect(err).NotTo(HaveOccurred())
				Expect(nextTime).NotTo(BeZero())

				loc, _ := time.LoadLocation("America/Los_Angeles")
				nextInLoc := nextTime.In(loc)
				Expect(nextInLoc.Hour()).To(Equal(9))
			})

			It("should calculate next time in Europe/London timezone", func() {
				ps := &powercapv1.PowercapSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-tz-london",
						Namespace: "default",
					},
					Spec: powercapv1.PowercapScheduleSpec{
						TimeZone: "Europe/London",
						Schedules: []powercapv1.PowercapRule{
							{
								Name:     "test-schedule",
								Schedule: "30 14 * * *",
								PowerLimits: []powercapv1.PowerLimitEntry{
									{
										Zone:       "intel-rapl:0",
										Constraint: "constraint_0",
										PowerLimit: 65000000,
									},
								},
							},
						},
					},
				}

				nextTime, err := reconciler.getNextScheduleTime(ps)
				Expect(err).NotTo(HaveOccurred())
				Expect(nextTime).NotTo(BeZero())

				loc, _ := time.LoadLocation("Europe/London")
				nextInLoc := nextTime.In(loc)
				Expect(nextInLoc.Hour()).To(Equal(14))
				Expect(nextInLoc.Minute()).To(Equal(30))
			})
		})

		Context("with UTC timezone (default)", func() {
			It("should use UTC when timezone is empty", func() {
				ps := &powercapv1.PowercapSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-utc-default",
						Namespace: "default",
					},
					Spec: powercapv1.PowercapScheduleSpec{
						TimeZone: "",
						Schedules: []powercapv1.PowercapRule{
							{
								Name:     "test-schedule",
								Schedule: "0 12 * * *",
								PowerLimits: []powercapv1.PowerLimitEntry{
									{
										Zone:       "intel-rapl:0",
										Constraint: "constraint_0",
										PowerLimit: 65000000,
									},
								},
							},
						},
					},
				}

				nextTime, err := reconciler.getNextScheduleTime(ps)
				Expect(err).NotTo(HaveOccurred())
				Expect(nextTime).NotTo(BeZero())

				nextInUTC := nextTime.UTC()
				Expect(nextInUTC.Hour()).To(Equal(12))
			})

			It("should use UTC when timezone is explicitly UTC", func() {
				ps := &powercapv1.PowercapSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-utc-explicit",
						Namespace: "default",
					},
					Spec: powercapv1.PowercapScheduleSpec{
						TimeZone: "UTC",
						Schedules: []powercapv1.PowercapRule{
							{
								Name:     "test-schedule",
								Schedule: "0 15 * * *",
								PowerLimits: []powercapv1.PowerLimitEntry{
									{
										Zone:       "intel-rapl:0",
										Constraint: "constraint_0",
										PowerLimit: 65000000,
									},
								},
							},
						},
					},
				}

				nextTime, err := reconciler.getNextScheduleTime(ps)
				Expect(err).NotTo(HaveOccurred())
				Expect(nextTime).NotTo(BeZero())

				nextInUTC := nextTime.UTC()
				Expect(nextInUTC.Hour()).To(Equal(15))
			})
		})

		Context("with invalid timezone", func() {
			It("should return ErrInvalidTimezone for unknown timezone", func() {
				ps := &powercapv1.PowercapSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-invalid-tz",
						Namespace: "default",
					},
					Spec: powercapv1.PowercapScheduleSpec{
						TimeZone: "Invalid/Timezone",
						Schedules: []powercapv1.PowercapRule{
							{
								Name:     "test-schedule",
								Schedule: "0 9 * * *",
								PowerLimits: []powercapv1.PowerLimitEntry{
									{
										Zone:       "intel-rapl:0",
										Constraint: "constraint_0",
										PowerLimit: 65000000,
									},
								},
							},
						},
					},
				}

				_, err := reconciler.getNextScheduleTime(ps)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidTimezone)).To(BeTrue())
			})

			It("should return ErrInvalidTimezone for malformed timezone string", func() {
				ps := &powercapv1.PowercapSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-malformed-tz",
						Namespace: "default",
					},
					Spec: powercapv1.PowercapScheduleSpec{
						TimeZone: "not-a-timezone",
						Schedules: []powercapv1.PowercapRule{
							{
								Name:     "test-schedule",
								Schedule: "0 9 * * *",
								PowerLimits: []powercapv1.PowerLimitEntry{
									{
										Zone:       "intel-rapl:0",
										Constraint: "constraint_0",
										PowerLimit: 65000000,
									},
								},
							},
						},
					},
				}

				_, err := reconciler.getNextScheduleTime(ps)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidTimezone)).To(BeTrue())
			})
		})

		Context("with multiple schedules", func() {
			It("should return the earliest next schedule time", func() {
				ps := &powercapv1.PowercapSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-multi-schedule",
						Namespace: "default",
					},
					Spec: powercapv1.PowercapScheduleSpec{
						TimeZone: "UTC",
						Schedules: []powercapv1.PowercapRule{
							{
								Name:     "later-schedule",
								Schedule: "0 23 * * *",
								PowerLimits: []powercapv1.PowerLimitEntry{
									{
										Zone:       "intel-rapl:0",
										Constraint: "constraint_0",
										PowerLimit: 30000000,
									},
								},
							},
							{
								Name:     "earlier-schedule",
								Schedule: "0 1 * * *",
								PowerLimits: []powercapv1.PowerLimitEntry{
									{
										Zone:       "intel-rapl:0",
										Constraint: "constraint_0",
										PowerLimit: 65000000,
									},
								},
							},
						},
					},
				}

				nextTime, err := reconciler.getNextScheduleTime(ps)
				Expect(err).NotTo(HaveOccurred())
				Expect(nextTime).NotTo(BeZero())

				nextInUTC := nextTime.UTC()
				Expect(nextInUTC.Hour()).To(BeElementOf(1, 23))
			})
		})

		Context("with no schedules", func() {
			It("should return error when schedules are empty", func() {
				ps := &powercapv1.PowercapSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-no-schedules",
						Namespace: "default",
					},
					Spec: powercapv1.PowercapScheduleSpec{
						TimeZone:  "UTC",
						Schedules: []powercapv1.PowercapRule{},
					},
				}

				_, err := reconciler.getNextScheduleTime(ps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no schedules defined"))
			})
		})

		Context("with invalid cron expression", func() {
			It("should return error for invalid cron syntax", func() {
				ps := &powercapv1.PowercapSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-invalid-cron",
						Namespace: "default",
					},
					Spec: powercapv1.PowercapScheduleSpec{
						TimeZone: "UTC",
						Schedules: []powercapv1.PowercapRule{
							{
								Name:     "invalid-schedule",
								Schedule: "invalid cron",
								PowerLimits: []powercapv1.PowerLimitEntry{
									{
										Zone:       "intel-rapl:0",
										Constraint: "constraint_0",
										PowerLimit: 65000000,
									},
								},
							},
						},
					},
				}

				_, err := reconciler.getNextScheduleTime(ps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to parse schedule"))
			})
		})
	})
})
