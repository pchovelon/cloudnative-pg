/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
	"github.com/cloudnative-pg/cloudnative-pg/tests"
	"github.com/cloudnative-pg/cloudnative-pg/tests/utils/clusterutils"
	"github.com/cloudnative-pg/cloudnative-pg/tests/utils/exec"
	"github.com/cloudnative-pg/cloudnative-pg/tests/utils/yaml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PGBouncer Connections", Label(tests.LabelServiceConnectivity), func() {
	const (
		sampleFile                    = fixturesDir + "/pgbouncer/cluster-pgbouncer.yaml.template"
		poolerBasicAuthRWSampleFile   = fixturesDir + "/pgbouncer/pgbouncer-pooler-basic-auth-rw.yaml"
		poolerCertificateRWSampleFile = fixturesDir + "/pgbouncer/pgbouncer-pooler-tls-rw.yaml"
		poolerBasicAuthROSampleFile   = fixturesDir + "/pgbouncer/pgbouncer-pooler-basic-auth-ro.yaml"
		poolerCertificateROSampleFile = fixturesDir + "/pgbouncer/pgbouncer-pooler-tls-ro.yaml"
		level                         = tests.Low
	)
	var err error
	var namespace, clusterName string
	BeforeEach(func() {
		if testLevelEnv.Depth < int(level) {
			Skip("Test depth is lower than the amount requested for this test")
		}
	})

	Context("no user-defined certificates", Ordered, func() {
		BeforeAll(func() {
			// Create a cluster in a namespace we'll delete after the test
			namespace, err = env.CreateUniqueTestNamespace(env.Ctx, env.Client, "pgbouncer-auth-no-user-certs")
			Expect(err).ToNot(HaveOccurred())
			clusterName, err = yaml.GetResourceNameFromYAML(env.Scheme, sampleFile)
			Expect(err).ToNot(HaveOccurred())
			AssertCreateCluster(namespace, clusterName, sampleFile, env)
		})
		JustAfterEach(func() {
			primaryPod, err := clusterutils.GetPrimary(env.Ctx, env.Client, namespace, clusterName)
			Expect(err).ToNot(HaveOccurred())
			DeleteTableUsingPgBouncerService(namespace, clusterName, poolerBasicAuthRWSampleFile, env, primaryPod)
		})

		It("can connect to Postgres via pgbouncer service using basic authentication", func() {
			By("setting up read write type pgbouncer pooler", func() {
				createAndAssertPgBouncerPoolerIsSetUp(namespace, poolerBasicAuthRWSampleFile, 1)
				assertPgBouncerPoolerDeploymentStrategy(namespace, poolerBasicAuthRWSampleFile, "25%", "25%")
			})

			By("setting up read only type pgbouncer pooler", func() {
				createAndAssertPgBouncerPoolerIsSetUp(namespace, poolerBasicAuthROSampleFile, 1)
				assertPgBouncerPoolerDeploymentStrategy(namespace, poolerBasicAuthROSampleFile, "24%", "24%")
			})

			By("verifying read and write connections using pgbouncer service", func() {
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerBasicAuthRWSampleFile, true)
			})

			By("verifying read connections using pgbouncer service", func() {
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerBasicAuthROSampleFile, false)
			})

			By("executing psql within the pgbouncer pod", func() {
				pod, err := getPgbouncerPod(namespace, poolerBasicAuthRWSampleFile)
				Expect(err).ToNot(HaveOccurred())

				err = runShowHelpInPod(pod)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("can connect to Postgres via pgbouncer service using tls certificates", func() {
			By("setting up read write type pgbouncer pooler", func() {
				createAndAssertPgBouncerPoolerIsSetUp(namespace, poolerCertificateRWSampleFile, 1)
			})

			By("setting up read only type pgbouncer pooler", func() {
				createAndAssertPgBouncerPoolerIsSetUp(namespace, poolerCertificateROSampleFile, 1)
			})

			By("verifying read and write connections using pgbouncer service", func() {
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerCertificateRWSampleFile, true)
			})

			By("verifying read connections using pgbouncer service", func() {
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerCertificateROSampleFile, false)
			})
		})

		It("should recreate after deleting pgbouncer pod", func() {
			assertPodIsRecreated(namespace, poolerBasicAuthRWSampleFile)
			By("verifying pgbouncer read write service connections after deleting pod", func() {
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerBasicAuthRWSampleFile, true)
			})

			assertPodIsRecreated(namespace, poolerBasicAuthROSampleFile)
			By("verifying pgbouncer read only service connections after pod deleting", func() {
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerBasicAuthROSampleFile, false)
			})
		})

		It("should recreate after deleting pgbouncer deployment", func() {
			assertDeploymentIsRecreated(namespace, poolerBasicAuthRWSampleFile)
			By("verifying pgbouncer read write service connections after deleting deployment", func() {
				// verify read and write connections after pgbouncer deployment deletion
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerBasicAuthRWSampleFile, true)
			})

			assertDeploymentIsRecreated(namespace, poolerBasicAuthROSampleFile)
			By("verifying pgbouncer read only service connections after deleting deployment", func() {
				// verify read and write connections after pgbouncer deployment deletion
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerBasicAuthROSampleFile, false)
			})
		})
	})

	Context("user-defined certificates", func() {
		It("can connect to Postgres via pgbouncer using different client and server CA", func() {
			const (
				folderPath                    = fixturesDir + "/pgbouncer/pgbouncer_separate_client_server_ca/"
				sampleFileWithCertificate     = folderPath + "cluster-user-supplied-client-server-certificates.yaml.template"
				poolerCertificateROSampleFile = folderPath + "pgbouncer-pooler-tls-ro.yaml"
				poolerCertificateRWSampleFile = folderPath + "pgbouncer-pooler-tls-rw.yaml"
				caSecName                     = "my-postgresql-server-ca"
				tlsSecName                    = "my-postgresql-server"
				tlsSecNameClient              = "my-postgresql-client"
				caSecNameClient               = "my-postgresql-client-ca"
			)
			// Create a cluster in a namespace that will be deleted after the test
			namespace, err = env.CreateUniqueTestNamespace(env.Ctx, env.Client, "pgbouncer-separate-certificates")
			Expect(err).ToNot(HaveOccurred())
			clusterName, err = yaml.GetResourceNameFromYAML(env.Scheme, sampleFileWithCertificate)
			Expect(err).ToNot(HaveOccurred())

			// Create certificates secret for server
			CreateAndAssertServerCertificatesSecrets(namespace, clusterName,
				caSecName, tlsSecName, true)

			// Create certificates secret for client
			CreateAndAssertClientCertificatesSecrets(namespace, clusterName,
				caSecNameClient, tlsSecNameClient, "app-user-cert", true)

			AssertCreateCluster(namespace, clusterName, sampleFileWithCertificate, env)

			By("setting up read write type pgbouncer pooler", func() {
				createAndAssertPgBouncerPoolerIsSetUp(namespace, poolerCertificateRWSampleFile, 1)
			})

			By("setting up read only type pgbouncer pooler", func() {
				createAndAssertPgBouncerPoolerIsSetUp(namespace, poolerCertificateROSampleFile, 1)
			})

			By("verifying read and write connections using pgbouncer service", func() {
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerCertificateRWSampleFile, true)
			})

			By("verifying read connections using pgbouncer service", func() {
				assertReadWriteConnectionUsingPgBouncerService(namespace, clusterName,
					poolerCertificateROSampleFile, false)
			})
		})
	})
})

func getPgbouncerPod(namespace, sampleFile string) (*corev1.Pod, error) {
	poolerKey, err := yaml.GetResourceNameFromYAML(env.Scheme, sampleFile)
	if err != nil {
		return nil, err
	}

	Expect(err).ToNot(HaveOccurred())

	var podList corev1.PodList
	err = env.Client.List(env.Ctx, &podList, ctrlclient.InNamespace(namespace),
		ctrlclient.MatchingLabels{utils.PgbouncerNameLabel: poolerKey})
	Expect(err).ToNot(HaveOccurred())
	Expect(len(podList.Items)).Should(BeEquivalentTo(1))
	return &podList.Items[0], nil
}

func runShowHelpInPod(pod *corev1.Pod) error {
	_, _, err := exec.Command(
		env.Ctx, env.Interface, env.RestClientConfig, *pod,
		"pgbouncer", nil, "psql", "-c", "SHOW HELP",
	)
	return err
}
