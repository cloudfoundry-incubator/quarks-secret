package kube_test

import (
	b64 "encoding/base64"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/quarks-secret/testing"
	cmdHelper "code.cloudfoundry.org/quarks-utils/testing"
)

var _ = Describe("Examples Directory", func() {
	var (
		example      string
		yamlFilePath string
		kubectl      *cmdHelper.Kubectl
	)

	JustBeforeEach(func() {
		kubectl = cmdHelper.NewKubectl()
		yamlFilePath = path.Join(example)
		err := cmdHelper.Create(namespace, yamlFilePath)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("rotation example", func() {
		var (
			passwordv1 []byte
			passwordv2 []byte
		)

		When("rotation config lists one quarks secret", func() {
			BeforeEach(func() {
				example = filepath.Join(examplesDir, "password.yaml")
			})

			It("should change the password data", func() {
				By("Wating for the password secret")
				err := kubectl.WaitForSecret(namespace, "gen-secret1")
				Expect(err).ToNot(HaveOccurred())
				passwordv1, err = cmdHelper.GetData(namespace, "secret", "gen-secret1", "go-template={{.data.password}}")
				Expect(err).ToNot(HaveOccurred())
				Expect(passwordv1).NotTo(BeNil())

				By("Creating the rotate configmap")
				example = "rotate.yaml"
				yamlFilePath = path.Join(examplesDir, example)
				err = cmdHelper.Create(namespace, yamlFilePath)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the rotated password data")
				passwordv1, err = cmdHelper.GetData(namespace, "secret", "gen-secret1", "go-template={{.data.password}}")
				Expect(err).ToNot(HaveOccurred())
				Expect(passwordv1).NotTo(Equal(passwordv2))
			})
		})
	})

	Context("user-provided example", func() {
		var (
			passwordv1 []byte
			passwordv2 []byte
		)

		When("creating an owning qsec", func() {
			BeforeEach(func() {
				example = filepath.Join(examplesDir, "user-provided-secret.yaml")
			})

			It("does not modify the user-provided secret", func() {
				By("Waiting for the password secret")
				err := kubectl.WaitForSecret(namespace, "gen-secret1")
				Expect(err).ToNot(HaveOccurred())
				passwordv1, err = cmdHelper.GetData(namespace, "secret", "gen-secret1", "go-template={{.data.password}}")
				Expect(err).ToNot(HaveOccurred())
				Expect(passwordv1).NotTo(BeNil())

				By("Creating the owning QuarksSecrets")
				err = cmdHelper.Create(namespace, filepath.Join(examplesDir, "password.yaml"))
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					generated, err := cmdHelper.GetData(namespace, "secret", "gen-secret1", "go-template={{.status.generated}}")
					if err != nil {
						return false
					}
					return string(generated) == "true"
				})

				By("Checking the rotated password data")
				passwordv2, err = cmdHelper.GetData(namespace, "secret", "gen-secret1", "go-template={{.data.password}}")
				Expect(err).ToNot(HaveOccurred())
				Expect(passwordv1).To(Equal(passwordv2))
			})
		})
	})

	Context("quarks-secret copies", func() {
		var copyNamespace string
		var tempQSecretFileName string

		BeforeEach(func() {
			copyNamespace = "qseccopy-" + strconv.Itoa(int(nsIndex)) + "-" +
				strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

			err := cmdHelper.CreateNamespace(copyNamespace)
			Expect(err).ToNot(HaveOccurred())

			// Create a secret in the copy namespace

			// Create a copy of the example files with the correct namespaces in them
			dSecretExample := path.Join(examplesDir, "copy-secret-destination.yaml")
			dSecret, err := ioutil.ReadFile(dSecretExample)
			Expect(err).ToNot(HaveOccurred())
			tmpDSecret, err := ioutil.TempFile(os.TempDir(), "dsecret-*")
			defer os.Remove(tmpDSecret.Name())
			Expect(err).ToNot(HaveOccurred(), "creating tmp file")
			_, err = tmpDSecret.WriteString(
				strings.ReplaceAll(
					strings.ReplaceAll(
						string(dSecret), "COPYNAMESPACE", copyNamespace,
					), "NAMESPACE", namespace))
			Expect(err).ToNot(HaveOccurred())
			Expect(tmpDSecret.Close()).ToNot(HaveOccurred())

			// A copy of the QuarkSecret with the correct COPYNAMESPACE in it
			quarksSecretExample := path.Join(examplesDir, "copies.yaml")
			qSecret, err := ioutil.ReadFile(quarksSecretExample)
			Expect(err).ToNot(HaveOccurred())
			tmpQSecret, err := ioutil.TempFile(os.TempDir(), "qsec-*")
			tempQSecretFileName = tmpQSecret.Name()
			Expect(err).ToNot(HaveOccurred(), "creating tmp file in examples dir")
			_, err = tmpQSecret.WriteString(
				strings.ReplaceAll(
					string(qSecret), "COPYNAMESPACE", copyNamespace,
				))
			Expect(err).ToNot(HaveOccurred())
			Expect(tmpQSecret.Close()).ToNot(HaveOccurred())

			// Create the destination secret
			err = cmdHelper.Create(copyNamespace, tmpDSecret.Name())
			Expect(err).ToNot(HaveOccurred())

			example = tempQSecretFileName
		})

		AfterEach(func() {
			err := cmdHelper.DeleteNamespace(copyNamespace)
			Expect(err).ToNot(HaveOccurred())

			err = os.Remove(tempQSecretFileName)
			Expect(err).ToNot(HaveOccurred())
		})

		It("are created if everything is setup correctly", func() {
			By("Checking the generated password")
			err := cmdHelper.SecretCheckData(copyNamespace, "copied-secret", ".data.password")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("quarks-secret copy type", func() {
		var copyNamespace string
		var tempQSecretFileName string

		BeforeEach(func() {
			quarksSecretExample := path.Join(examplesDir, "copy.yaml")

			copyNamespace = "qseccopy-" + strconv.Itoa(int(nsIndex)) + "-" +
				strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

			err := cmdHelper.CreateNamespace(copyNamespace)
			Expect(err).ToNot(HaveOccurred())

			// Create a copy of the example files with the correct namespaces in them
			dSecretExample := path.Join(examplesDir, "copy-qsecret-destination.yaml")
			dSecret, err := ioutil.ReadFile(dSecretExample)
			Expect(err).ToNot(HaveOccurred())
			tmpDSecret, err := ioutil.TempFile(os.TempDir(), "dsecret-*")
			defer os.Remove(tmpDSecret.Name())
			Expect(err).ToNot(HaveOccurred(), "creating tmp file")
			_, err = tmpDSecret.WriteString(
				strings.ReplaceAll(
					strings.ReplaceAll(
						string(dSecret), "COPYNAMESPACE", copyNamespace,
					), "NAMESPACE", namespace))
			Expect(err).ToNot(HaveOccurred())
			Expect(tmpDSecret.Close()).ToNot(HaveOccurred())

			// Create a secret in the copy namespace
			qSecret, err := ioutil.ReadFile(quarksSecretExample)
			Expect(err).ToNot(HaveOccurred())
			tmpQSecret, err := ioutil.TempFile(os.TempDir(), "qsec-*")
			tempQSecretFileName = tmpQSecret.Name()
			Expect(err).ToNot(HaveOccurred(), "creating tmp file in examples dir")
			_, err = tmpQSecret.WriteString(
				strings.ReplaceAll(
					string(qSecret), "COPYNAMESPACE", copyNamespace,
				))
			Expect(err).ToNot(HaveOccurred())
			Expect(tmpQSecret.Close()).ToNot(HaveOccurred())

			//Create the destination secret
			err = cmdHelper.Create(copyNamespace, tmpDSecret.Name())
			Expect(err).ToNot(HaveOccurred())

			example = tempQSecretFileName

		})

		AfterEach(func() {

			err := cmdHelper.DeleteNamespace(copyNamespace)
			Expect(err).ToNot(HaveOccurred())

			err = os.Remove(example)
			Expect(err).ToNot(HaveOccurred())
		})

		It("are created if everything is setup correctly", func() {
			By("Checking the generated secrets")
			err := kubectl.WaitForSecret(namespace, "gen-secret")
			Expect(err).ToNot(HaveOccurred(), "error waiting for secret")
			By("Checking the copied secrets")

			err = kubectl.WaitForSecret(copyNamespace, "copied-secret")
			Expect(err).ToNot(HaveOccurred(), "error waiting for secret")
			By("Checking the copied secrets contents")

			err = cmdHelper.SecretCheckData(copyNamespace, "copied-secret", ".data.password")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("API server signed certificate example", func() {
		BeforeEach(func() {
			example = filepath.Join(examplesDir, "certificate.yaml")
		})

		It("creates a signed cert", func() {
			By("Checking the generated certificate")
			err := kubectl.WaitForSecret(namespace, "gen-certificate")
			Expect(err).ToNot(HaveOccurred(), "error waiting for secret")
			err = cmdHelper.SecretCheckData(namespace, "gen-certificate", ".data.certificate")
			Expect(err).ToNot(HaveOccurred(), "error getting for secret")
		})
	})

	Context("self signed certificate example", func() {
		BeforeEach(func() {
			example = filepath.Join(examplesDir, "loggregator-ca-cert.yaml")
		})

		It("creates a self-signed certificate", func() {
			certYamlFilePath := examplesDir + "loggregator-tls-agent-cert.yaml"

			By("Creating QuarksSecrets")
			err := cmdHelper.Create(namespace, certYamlFilePath)
			Expect(err).ToNot(HaveOccurred())

			By("Checking the generated certificates")
			err = kubectl.WaitForSecret(namespace, "example.var-loggregator-ca")
			Expect(err).ToNot(HaveOccurred(), "error waiting for ca secret")
			err = kubectl.WaitForSecret(namespace, "example.var-loggregator-tls-agent")
			Expect(err).ToNot(HaveOccurred(), "error waiting for cert secret")

			By("Checking the generated certificates")
			outSecret, err := cmdHelper.GetData(namespace, "secret", "example.var-loggregator-ca", "go-template={{.data.certificate}}")
			Expect(err).ToNot(HaveOccurred())
			rootPEM, _ := b64.StdEncoding.DecodeString(string(outSecret))

			outSecret, err = cmdHelper.GetData(namespace, "secret", "example.var-loggregator-tls-agent", "go-template={{.data.certificate}}")
			Expect(err).ToNot(HaveOccurred())
			certPEM, _ := b64.StdEncoding.DecodeString(string(outSecret))

			By("Verify the certificates")
			dnsName := "metron"
			err = testing.CertificateVerify(rootPEM, certPEM, dnsName)
			Expect(err).ToNot(HaveOccurred(), "error verifying certificates")
		})
	})

	Context("rsa keys example", func() {
		var (
			privateKey []byte
			publicKey  []byte
		)

		BeforeEach(func() {
			example = filepath.Join(examplesDir, "rsa.yaml")
		})

		It("should generate the rsa keys data", func() {
			By("Creating the rsa secret")
			expectedSecretName := "rsa-keys-1"
			err := kubectl.WaitForSecret(namespace, expectedSecretName)
			Expect(err).ToNot(HaveOccurred())
			privateKey, err = cmdHelper.GetData(namespace, "secret", expectedSecretName, "go-template={{.data.private_key}}")
			Expect(err).ToNot(HaveOccurred())
			publicKey, err = cmdHelper.GetData(namespace, "secret", expectedSecretName, "go-template={{.data.public_key}}")
			Expect(err).ToNot(HaveOccurred())
			Expect(privateKey).NotTo(BeNil())
			Expect(publicKey).NotTo(BeNil())
		})
	})

	Context("dockerConfigJson secret example", func() {
		BeforeEach(func() {
			example = filepath.Join(examplesDir, "docker-registry-secret.yaml")
		})

		It("creates a dockerConfigJson secret", func() {
			By("Checking the generated data")
			err := kubectl.WaitForSecret(namespace, "gen-docker-registry-secret")
			Expect(err).ToNot(HaveOccurred(), "error waiting for secret")
			err = cmdHelper.SecretCheckData(namespace, "gen-docker-registry-secret", "index .data  \".dockerconfigjson\"")
			Expect(err).ToNot(HaveOccurred(), "error getting for secret")
		})
	})

	Context("tls type certificate example", func() {
		BeforeEach(func() {
			example = filepath.Join(examplesDir, "ca.yaml")
		})

		It("creates a signed certificate", func() {
			certYamlFilePath := examplesDir + "tls.yaml"

			By("Creating QuarksSecrets")
			err := cmdHelper.Create(namespace, certYamlFilePath)
			Expect(err).ToNot(HaveOccurred())

			By("Checking the generated certificates")
			err = kubectl.WaitForSecret(namespace, "example.secret.ca")
			Expect(err).ToNot(HaveOccurred(), "error waiting for ca secret")
			err = kubectl.WaitForSecret(namespace, "example.secret.tls")
			Expect(err).ToNot(HaveOccurred(), "error waiting for cert secret")

			By("Checking the generated certificates")
			outSecret, err := cmdHelper.GetData(namespace, "secret", "example.secret.ca", "go-template={{.data.certificate}}")
			Expect(err).ToNot(HaveOccurred())
			rootPEM, _ := b64.StdEncoding.DecodeString(string(outSecret))

			outSecret, err = cmdHelper.GetData(namespace, "secret", "example.secret.tls", "go-template={{ index .data \"tls.crt\" }}")
			Expect(err).ToNot(HaveOccurred())
			certPEM, _ := b64.StdEncoding.DecodeString(string(outSecret))

			By("Verify the certificates")
			dnsName := "kubeTlsTypeCert"
			err = testing.CertificateVerify(rootPEM, certPEM, dnsName)
			Expect(err).ToNot(HaveOccurred(), "error verifying certificates")
		})
	})
})
