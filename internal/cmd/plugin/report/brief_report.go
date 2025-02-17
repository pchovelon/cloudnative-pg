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

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/internal/cmd/plugin"
)

// clusterReport contains the data to be printed by the `report cluster` plugin
type briefReport struct {
	clusterList cnpgv1.ClusterList
}

// cluster implements the "brief cluster" subcommand
// Produces a zip file containing
//   - cluster resources and specifications
func brief(ctx context.Context, format plugin.OutputFormat, file string) error {

	var clusters cnpgv1.ClusterList

	err := plugin.Client.List(ctx, &clusters)

	if err != nil {
		return fmt.Errorf("could not get clusters: %w", err)
	}

	for _, cluster := range clusters.Items {
		fmt.Println("PostgreSQL cluster : " + cluster.Name)
		fmt.Println("Namespace : " + cluster.Namespace)
		fmt.Println("Image : " + cluster.Spec.ImageName)
		fmt.Println("Instance(s) :")
		for _, instance := range cluster.Status.InstanceNames {
			fmt.Println("  " + instance)
		}
		fmt.Println("Configuration :")
		for _, parameter := range cluster.Spec.PostgresConfiguration.Parameters {
			fmt.Println("  " + parameter
		}

	}

	return nil
}
