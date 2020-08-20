# Contributing

## Dev Guide

### Running test
* Unit tests: `make test-unit` (see [Makefile](/Makefile))  
* Integration tests: `make test-cluster` (see [Makefile](/Makefile))  
    * Pre-requisite: A locally running k8s cluster accessible via `kubectl`. 
    You can use [Minikube](https://kubernetes.io/docs/setup/learning-environment/minikube/) (with the command `minikube start`)
    or [kind](https://kind.sigs.k8s.io/) to start a local cluster.
* Other tests: End-to-end tests for QuarksSecret examples 
    * [These tests](/e2e/kube) are still incomplete and under development. They use helm to install `QuarksSecret`
    onto a running k8s cluster and apply the QuarksSecret examples included in the [/docs/examples](/docs/examples) directory.
    * Pre-requisites:
        * A locally running k8s cluster accessible via `kubectl` (same as the integration tests).
        * Edit the values in the [helm chart values file](/deploy/helm/quarks-secret/values.yaml) 
        to specify a `QuarksSecret` image.
