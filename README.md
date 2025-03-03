# cluster-api-provider-wxone

The Kubernetes Cluster API Provider WXOne is an infrastructure provider for the Kubernetes Cluster API project (https://cluster-api.sigs.k8s.io/)

## Local development

#### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- kind (https://kind.sigs.k8s.io/)
- clusterctl (https://cluster-api.sigs.k8s.io/user/quick-start#install-clusterctl)
- tilt (https://docs.tilt.dev/install.html)
- checkout cluster api (https://github.com/kubernetes-sigs/cluster-api) in the same directory!
- checkout cluster api provider rke2 (https://github.com/rancher/cluster-api-provider-rke2) in the same directory!

#### Develop with tilt (recommended)

- create a flavor of type `vs1` with name `vs1 monthly`:
- create an image with name `local`
- change reg_port in ../cluster-api/hack/kind-install-for-capd to 5002 so that local registry does not collide with api

```sh
# setup

cat > ../cluster-api/tilt-settings.yaml <<EOF
default_registry: ""
provider_repos:
  - ../cluster-api-provider-wxone
  - ../cluster-api-provider-rke2
enable_providers:
  - wxone
  - rke2-control-plane
  - rke2-bootstrap
template_dirs:
  wxone: 
    - ../cluster-api-provider-wxone/templates
kustomize_substitutions:
  KUBERNETES_VERSION: "v1.30.2+rke2r1"
EOF

# run

cd ../cluster-api
make tilt-up
```

- hit space to open tilt in browser
- under CAPW.templates a new cluster can be deployed

```
# clean up

make clean-kind
make clean
```

#### Develop manually

```sh
cat > kind-cluster-with-extramounts.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: capi-test
nodes:
- role: control-plane
extraMounts:
    - hostPath: /var/run/docker.sock
    containerPath: /var/run/docker.sock
EOF
kind create cluster --config kind-cluster-with-extramounts.yaml`
# clusterctl init
clusterctl init --core cluster-api:v1.9.0 --bootstrap rke2:v0.10.0 --control-plane rke2:v0.10.0 --infrastructure docker:v1.9.0
make docker-build
kind load docker-image controller:latest --name capi-test

export WX_ONE_HOST=http://host.docker.internal:5000
# fill in username of wx one api key
export WX_ONE_USERNAME=xxx
# fill in password of wx one api key
export WX_ONE_PASSWORD=xxx

make install deploy


kubectl apply -f examples/kind-rke2-cluster.yaml
```

#### Add new graphql queries/mutations or update schema

- for schema updates update file `internal/controller/schema.graphql` (it contains `customerSchema.gql` concatenated with `commonSchema.gql`)
  ```
  cat ../api-gateway/lib/graphql/commonSchema.gql > internal/controller/schema.graphql && \
  echo "" >> internal/controller/schema.graphql && \
  cat ../api-gateway/lib/graphql/customerSchema.gql >> internal/controller/schema.graphql
  ```
- to add or update queries and mutations add them to the file `internal/controller/genqlient.graphql`
- to add additional go bindings for types update `internal/controller/genqlient.yaml` (for a complete list of configuration options see https://github.com/Khan/genqlient/blob/main/docs/genqlient.yaml)

after updating run

```
cd internal/controller
go run github.com/Khan/genqlient
```

## Getting Started

### Prerequisites
- go version v1.23.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/cluster-api-provider-wxone:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/cluster-api-provider-wxone:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/cluster-api-provider-wxone:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/cluster-api-provider-wxone/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
kubebuilder edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

## Contributing


**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

