module code.cloudfoundry.org/quarks-secret

go 1.15

require (
	code.cloudfoundry.org/quarks-utils v0.0.2-0.20201023094315-093d282bd4ab
	github.com/cloudflare/cfssl v1.4.1
	github.com/dchest/uniuri v0.0.0-20200228104902-7aecb25e1fe5
	github.com/go-logr/logr v0.2.0
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	helm.sh/helm/v3 v3.4.0
	k8s.io/api v0.19.2
	k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.6.3
)
