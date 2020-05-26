package kube_test

import (
	b64 "encoding/base64"
	"io/ioutil"
	"os"
	"path"
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
		yamlFilePath = path.Join(examplesDir, example)
		err := cmdHelper.Create(namespace, yamlFilePath)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("quarks-secret example", func() {
		BeforeEach(func() {
			example = "quarks-secret/password.yaml"
		})

		It("generates a password", func() {
			By("Checking the generated password")
			err := cmdHelper.SecretCheckData(namespace, "gen-secret1", ".data.password")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("quarks-secret copies", func() {
		var copyNamespace string
		var tempQSecretFileName string

		BeforeEach(func() {
			// example = "quarks-secret/copies.yaml"
			copyNamespace = "qseccopy-" + strconv.Itoa(int(nsIndex)) + "-" +
				strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

			cmdHelper.CreateNamespace(copyNamespace)
			// Create a secret in the copy namespace

			// Create a copy of the example files with the correct namespaces in them
			dSecretExample := path.Join(examplesDir, "quarks-secret/copy-secret-destination.yaml")
			dSecret, err := ioutil.ReadFile(dSecretExample)
			Expect(err).ToNot(HaveOccurred())
			tmpDSecret, err := ioutil.TempFile("", "dsecret-*")
			defer os.Remove(tmpDSecret.Name())
			Expect(err).ToNot(HaveOccurred())
			_, err = tmpDSecret.WriteString(
				strings.ReplaceAll(
					strings.ReplaceAll(
						string(dSecret), "COPYNAMESPACE", copyNamespace,
					), "NAMESPACE", namespace))
			Expect(err).ToNot(HaveOccurred())
			Expect(tmpDSecret.Close()).ToNot(HaveOccurred())

			// A copy of the QuarkSecret with the correct COPYNAMESPACE in it
			quarksSecretExample := path.Join(examplesDir, "quarks-secret/copies.yaml")
			qSecret, err := ioutil.ReadFile(quarksSecretExample)
			Expect(err).ToNot(HaveOccurred())
			tmpQSecret, err := ioutil.TempFile("", "qsec-*")
			tempQSecretFileName = tmpQSecret.Name()
			Expect(err).ToNot(HaveOccurred())
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
			cmdHelper.DeleteNamespace(copyNamespace)
			os.Remove(tempQSecretFileName)
		})

		It("are created if everything is setup correctly", func() {
			By("Checking the generated password")
			err := cmdHelper.SecretCheckData(copyNamespace, "copied-secret", ".data.password")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("API server signed certificate example", func() {
		BeforeEach(func() {
			example = "quarks-secret/certificate.yaml"
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
			example = "quarks-secret/loggregator-ca-cert.yaml"
		})

		It("creates a self-signed certificate", func() {
			certYamlFilePath := examplesDir + "quarks-secret/loggregator-tls-agent-cert.yaml"

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
})
