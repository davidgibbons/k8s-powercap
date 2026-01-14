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

package main

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

var _ = BeforeSuite(func() {
	zapLogger, _ := zap.NewDevelopment()
	logger = zapLogger.Sugar()
})

var _ = Describe("Agent Config", func() {
	Context("loadConfig", func() {
		BeforeEach(func() {
			os.Clearenv()
		})

		It("should parse valid SCHEDULES_JSON", func() {
			schedulesJSON := `[{"name":"test","schedule":"0 9 * * 1","powerLimits":[{"zone":"intel-rapl:0","constraint":"constraint_0","powerLimit":65000000}]}]`
			os.Setenv("SCHEDULES_JSON", schedulesJSON)
			os.Setenv("TIMEZONE", "UTC")
			os.Setenv("NAMESPACE", "default")
			os.Setenv("POD_NAME", "test-pod")

			config, err := loadConfig(logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).NotTo(BeNil())
			Expect(len(config.Schedules)).To(Equal(1))
			Expect(config.Schedules[0].Name).To(Equal("test"))
		})

		It("should error when SCHEDULES_JSON is missing", func() {
			_, err := loadConfig(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SCHEDULES_JSON"))
		})

		It("should error when SCHEDULES_JSON is invalid", func() {
			os.Setenv("SCHEDULES_JSON", "{invalid json}")

			_, err := loadConfig(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse"))
		})

		It("should error when schedule has no name", func() {
			schedulesJSON := `[{"schedule":"0 9 * * 1","powerLimits":[{"zone":"intel-rapl:0","constraint":"constraint_0","powerLimit":65000000}]}]`
			os.Setenv("SCHEDULES_JSON", schedulesJSON)

			_, err := loadConfig(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name is required"))
		})

		It("should error when schedule has no power limits", func() {
			schedulesJSON := `[{"name":"test","schedule":"0 9 * * 1","powerLimits":[]}]`
			os.Setenv("SCHEDULES_JSON", schedulesJSON)

			_, err := loadConfig(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("power limit"))
		})

		It("should error when power limit is missing zone", func() {
			schedulesJSON := `[{"name":"test","schedule":"0 9 * * 1","powerLimits":[{"constraint":"constraint_0","powerLimit":65000000}]}]`
			os.Setenv("SCHEDULES_JSON", schedulesJSON)

			_, err := loadConfig(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("zone"))
		})
	})
})
