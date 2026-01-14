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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	powercapv1 "github.com/davidgibbons/k8s-powercap/api/v1"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("PowercapSchedule Webhook", func() {
	var (
		obj       *powercapv1.PowercapSchedule
		oldObj    *powercapv1.PowercapSchedule
		validator PowercapScheduleCustomValidator
		defaulter PowercapScheduleCustomDefaulter
	)

	BeforeEach(func() {
		obj = &powercapv1.PowercapSchedule{}
		oldObj = &powercapv1.PowercapSchedule{}
		validator = PowercapScheduleCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = PowercapScheduleCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		// TODO (user): Add any setup logic common to all tests
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating PowercapSchedule under Defaulting Webhook", func() {
		// TODO (user): Add logic for defaulting webhooks
		// Example:
		// It("Should apply defaults when a required field is empty", func() {
		//     By("simulating a scenario where defaults should be applied")
		//     obj.SomeFieldWithDefault = ""
		//     By("calling the Default method to apply defaults")
		//     defaulter.Default(ctx, obj)
		//     By("checking that the default values are set")
		//     Expect(obj.SomeFieldWithDefault).To(Equal("default_value"))
		// })
	})

	Context("When creating or updating PowercapSchedule under Validating Webhook", func() {
		validPowerLimits := []powercapv1.PowerLimitEntry{
			{Zone: "intel-rapl:0", Constraint: "constraint_0", PowerLimit: 65000000},
		}

		It("Should accept valid cron schedules", func() {
			validSchedules := []string{
				"0 9 * * 1",
				"30 14 15 * *",
				"0 0 1 1 *",
				"* * * * *",
				"0 0 * * *",
				"59 23 31 12 6",
				"0 9 * * 1-5",      // M-F 9am (DOW range)
				"0 9 1-15 * *",     // 9am, first half of month (DOM range)
				"0 9 * 1-6 *",      // 9am, first half of year (month range)
				"0 9 1-15 1-6 1-5", // Combined ranges (DOM, month, DOW)
			}

			for _, sched := range validSchedules {
				obj.Spec.Schedules = []powercapv1.PowercapRule{
					{Name: "test", Schedule: sched, PowerLimits: validPowerLimits},
				}
				_, err := validator.ValidateCreate(ctx, obj)
				Expect(err).NotTo(HaveOccurred(), "Schedule %q should be valid", sched)
			}
		})

		It("Should reject comma-separated cron schedules", func() {
			invalidSchedules := []string{
				"0,30 9 * * *",
				"0 9,10,11 * * *",
				"0 9 1,15 * *",
				"0 9 * 1,6 *",
				"0 9 * * 1,2,3",
			}

			for _, sched := range invalidSchedules {
				obj.Spec.Schedules = []powercapv1.PowercapRule{
					{Name: "test", Schedule: sched, PowerLimits: validPowerLimits},
				}
				_, err := validator.ValidateCreate(ctx, obj)
				Expect(err).To(HaveOccurred(), "Schedule %q should be rejected (comma lists not allowed)", sched)
				Expect(err.Error()).To(ContainSubstring("invalid format"))
			}
		})

		It("Should reject step value cron schedules", func() {
			invalidSchedules := []string{
				"0 9 */2 * *",
				"0 */4 * * *",
				"*/15 * * * *",
			}

			for _, sched := range invalidSchedules {
				obj.Spec.Schedules = []powercapv1.PowercapRule{
					{Name: "test", Schedule: sched, PowerLimits: validPowerLimits},
				}
				_, err := validator.ValidateCreate(ctx, obj)
				Expect(err).To(HaveOccurred(), "Schedule %q should be rejected (step values not allowed)", sched)
				Expect(err.Error()).To(ContainSubstring("invalid format"))
			}
		})

		It("Should reject minute and hour range cron schedules", func() {
			invalidSchedules := []string{
				"0-30 9 * * *",    // minute range
				"0 9-17 * * *",    // hour range
				"0-59 0-23 * * *", // both minute and hour ranges
			}

			for _, sched := range invalidSchedules {
				obj.Spec.Schedules = []powercapv1.PowercapRule{
					{Name: "test", Schedule: sched, PowerLimits: validPowerLimits},
				}
				_, err := validator.ValidateCreate(ctx, obj)
				Expect(err).To(HaveOccurred(), "Schedule %q should be rejected (minute/hour ranges not allowed)", sched)
				Expect(err.Error()).To(ContainSubstring("minute and hour fields must be specific values"))
			}
		})

		It("Should reject schedules with no entries", func() {
			obj.Spec.Schedules = []powercapv1.PowercapRule{}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one schedule"))
		})

		It("Should reject duplicate schedule names", func() {
			obj.Spec.Schedules = []powercapv1.PowercapRule{
				{Name: "same-name", Schedule: "0 9 * * *", PowerLimits: validPowerLimits},
				{Name: "same-name", Schedule: "0 10 * * *", PowerLimits: validPowerLimits},
			}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate schedule name"))
		})
	})

})
