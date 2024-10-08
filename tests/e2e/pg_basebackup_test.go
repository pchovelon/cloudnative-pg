/*
Copyright The CloudNativePG Contributors

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

package e2e

import (
	apiv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/tests"
	"github.com/cloudnative-pg/cloudnative-pg/tests/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bootstrap with pg_basebackup", Label(tests.LabelRecovery), func() {
	const (
		namespacePrefix = "cluster-pg-basebackup"
		srcCluster      = fixturesDir + "/pg_basebackup/cluster-src.yaml.template"
		dstClusterBasic = fixturesDir + "/pg_basebackup/cluster-dst-basic-auth.yaml.template"
		dstClusterTLS   = fixturesDir + "/pg_basebackup/cluster-dst-tls.yaml.template"
		tableName       = "to_bootstrap"
		appUser         = "appuser"
		level           = tests.High
	)
	var namespace, srcClusterName string
	var err error
	BeforeEach(func() {
		if testLevelEnv.Depth < int(level) {
			Skip("Test depth is lower than the amount requested for this test")
		}
	})
	JustAfterEach(func() {
		if CurrentSpecReport().Failed() {
			env.DumpNamespaceObjects(namespace, "out/"+CurrentSpecReport().LeafNodeText+".log")
		}
	})

	Context("can bootstrap via pg_basebackup", Ordered, func() {
		BeforeAll(func() {
			// Create a cluster in a namespace we'll delete after the test
			namespace, err = env.CreateUniqueNamespace(namespacePrefix)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() error {
				return env.CleanupNamespace(
					namespace,
					CurrentSpecReport().LeafNodeText,
					CurrentSpecReport().Failed(),
					GinkgoWriter,
				)
			})
			// Create the source Cluster
			srcClusterName, err = env.GetResourceNameFromYAML(srcCluster)
			Expect(err).ToNot(HaveOccurred())
			AssertCreateCluster(namespace, srcClusterName, srcCluster, env)
			AssertCreateTestData(namespace, srcClusterName, tableName, psqlClientPod)
		})

		It("using basic authentication", func() {
			// Create the destination Cluster
			dstClusterName, err := env.GetResourceNameFromYAML(dstClusterBasic)
			Expect(err).ToNot(HaveOccurred())
			AssertCreateCluster(namespace, dstClusterName, dstClusterBasic, env)
			// We give more time than the usual 600s, since the recovery is slower
			AssertClusterIsReady(namespace, dstClusterName, testTimeouts[utils.ClusterIsReadySlow], env)

			secretName := dstClusterName + apiv1.ApplicationUserSecretSuffix

			By("checking the dst cluster with auto generated app password connectable", func() {
				AssertApplicationDatabaseConnection(namespace, dstClusterName,
					appUser, utils.AppDBName, "", secretName, psqlClientPod)
			})

			By("update user application password for dst cluster and verify connectivity", func() {
				const newPassword = "eeh2Zahohx" //nolint:gosec
				AssertUpdateSecret("password", newPassword, secretName, namespace, dstClusterName, 30, env)
				AssertApplicationDatabaseConnection(
					namespace,
					dstClusterName,
					appUser,
					utils.AppDBName,
					newPassword,
					secretName,
					psqlClientPod)
			})

			By("checking data have been copied correctly", func() {
				AssertDataExpectedCount(namespace, dstClusterName, tableName, 2, psqlClientPod)
			})

			By("writing some new data to the dst cluster", func() {
				insertRecordIntoTable(namespace, dstClusterName, tableName, 3, psqlClientPod)
			})

			By("checking the src cluster was not modified", func() {
				AssertDataExpectedCount(namespace, srcClusterName, tableName, 2, psqlClientPod)
			})
		})

		It("using TLS authentication", func() {
			// Create the destination Cluster
			dstClusterName, err := env.GetResourceNameFromYAML(dstClusterTLS)
			Expect(err).ToNot(HaveOccurred())
			AssertCreateCluster(namespace, dstClusterName, dstClusterTLS, env)
			// We give more time than the usual 600s, since the recovery is slower
			AssertClusterIsReady(namespace, dstClusterName, testTimeouts[utils.ClusterIsReadySlow], env)

			By("checking data have been copied correctly", func() {
				AssertDataExpectedCount(namespace, dstClusterName, tableName, 2, psqlClientPod)
			})

			By("writing some new data to the dst cluster", func() {
				insertRecordIntoTable(namespace, dstClusterName, tableName, 3, psqlClientPod)
			})

			By("checking the src cluster was not modified", func() {
				AssertDataExpectedCount(namespace, srcClusterName, tableName, 2, psqlClientPod)
			})
		})
	})
})
