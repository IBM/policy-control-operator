# policy-control-operator
This is a PoC for [the design proposal of "Policy Control for KCP/Edge"](https://docs.google.com/document/d/1pdqbZa53b9No5q49KVmsQtpMcgBTZcThseoKbdJSapY).

In the design, we introduce an controller who runs on a cluster in a service provider and enables policy control functions on both KCP workspace level and the managed clusters. The service cluster (policy control operator cluster) is also bound to a dedicated workspace so users who define policies just interact with KCP and don't need to interact with individual managed clusters. In this PoC, we use [Operator Framework](https://sdk.operatorframework.io/) and choose [Kyverno](https://kyverno.io/) the engine of policy control.

## Getting Started
You’ll need a Kubernetes cluster with ingress installed to run against. You can use [KIND](https://sigs.k8s.io/kind) and [KIND with Ingress](https://kind.sigs.k8s.io/docs/user/ingress/) to get a local cluster for testing, or run against a remote cluster.

**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/policy-control-operator:tag
```
	
2. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/policy-control-operator:tag KUSTOMIZE_VERSION=v4.5.7
```

3. Fill out [a Policy Control CR](./config/samples/pccr-template.yaml) and create the CR. Note that we assume KCP is running and one workspace is synced with a physical cluster. For example,

```sh
kubectl apply -f config/samples/pccr-edge1.yaml
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022 IBM Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
