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

package report

import (
	"context"
	"fmt"
	"strings"

	"github.com/cheynewallace/tabby"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/internal/cmd/plugin"
	"github.com/logrusorgru/aurora/v4"
)

// clusterReport contains the data to be printed by the `report cluster` plugin
type briefReport struct {
	clusterList cnpgv1.ClusterList
}

// brief implements the "brief" command
// Produces an output containing
//   - all clusters resources and their specifications

func brief(ctx context.Context, format plugin.OutputFormat, file string) error {

	fmt.Println(aurora.Green("Brief Report"))
	fmt.Println()

	fmt.Println(aurora.Green("Clusters Summary"))

	clustersSummary := tabby.New()

	clustersSummary.AddHeader(
		"PostgreSQL cluster",
		"Namespace",
		"Image",
		"Status",
		"Instances")

	var clusters cnpgv1.ClusterList
	err := plugin.Client.List(ctx, &clusters)
	if err != nil {
		return fmt.Errorf("could not get clusters: %w", err)
	}
	for _, cluster := range clusters.Items {
		clustersSummary.AddLine(cluster.Name, cluster.Namespace, cluster.Spec.ImageName, cluster.Status.Phase, strings.Join(cluster.Status.InstanceNames, ","))
		clustersSummary.Print()

		fmt.Println()
		for parameter, value := range cluster.Spec.PostgresConfiguration.Parameters {
			fmt.Println(parameter + " : " + value)
		}
	}

	fmt.Println()
	fmt.Println(aurora.Green("Operators Summary"))
	fmt.Println()

	operatorsSummary := tabby.New()

	operatorsSummary.AddHeader(
		"Operator name",
		"Namespace",
		"Image",
		"Status")

	return nil
}

// // briefCluster implements the "brief cluster" subcommand
// // Produces an output containing
// //   - one cluster resource and its informations

// func briefCluster(ctx context.Context, format plugin.OutputFormat, file string, cluster string) error {

// 	// fmt.Println("Configuration :")
// 	// for parameter, value := range cluster.Spec.PostgresConfiguration.Parameters {
// 	// 	fmt.Println(" " + parameter + " : " + value)
// 	// }

// 	return nil
// }
