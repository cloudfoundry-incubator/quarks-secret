module code.cloudfoundry.org/quarks-secret

go 1.15

require (
	code.cloudfoundry.org/quarks-utils v0.0.2
	github.com/cloudflare/cfssl v1.4.1
	github.com/dchest/uniuri v0.0.0-20200228104902-7aecb25e1fe5
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	helm.sh/helm/v3 v3.3.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.2
)
