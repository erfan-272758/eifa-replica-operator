# eifa-replica-operator


## Description
The `eifa-replica-operator` is a Kubernetes operator designed to dynamically manage the replica count of a deployment based on the output of a scheduled job. It utilizes a `JobTemplate` and a schedule spec (such as a cron expression) to execute jobs. After each job run, the operator reads the number from the last line of the job's log output and uses it to adjust the `replica` count of the target deployment (`scaleTargetRef`) accordingly.

This allows for dynamic scaling of services based on custom logic derived from the job's log output.


## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
If you prefer not to use `make` or want a simpler installation method, you can directly apply the generated manifest files:

### Install the Operator

You can install the operator by applying the bundled manifests file:

```sh
kubectl apply -f https://raw.githubusercontent.com/erfan-272758/eifa-replica-operator/refs/heads/main/config/all.manifests.yaml
```

This file includes all necessary components such as CRDs, roles, role bindings, service account, and the operator deployment.

### Install Sample Custom Resources

To test out the operator with example resources, you can apply the sample manifests located in the `example` directory:

```sh
kubectl apply -f example/random-replica/manifest.yaml
```

This will create sample custom resources that the operator can act upon. Make sure these examples are configured with appropriate default values.


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

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/eifa-replica-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/eifa-replica-operator/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025 Erfan Mahvash.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

