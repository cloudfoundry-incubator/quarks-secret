package quarkssecret

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"code.cloudfoundry.org/quarks-secret/pkg/credsgen"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
)

func (r *ReconcileQuarksSecret) createPasswordSecret(ctx context.Context, qsec *qsv1a1.QuarksSecret) error {
	request := credsgen.PasswordGenerationRequest{}
	password := r.generator.GeneratePassword(qsec.GetName(), request)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        qsec.Spec.SecretName,
			Namespace:   qsec.GetNamespace(),
			Labels:      qsec.Spec.SecretLabels,
			Annotations: qsec.Spec.SecretAnnotations,
		},
		StringData: map[string]string{
			"password": password,
		},
	}

	return r.createSecrets(ctx, qsec, secret)
}

func (r *ReconcileQuarksSecret) createRSASecret(ctx context.Context, qsec *qsv1a1.QuarksSecret) error {
	key, err := r.generator.GenerateRSAKey(qsec.GetName())
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        qsec.Spec.SecretName,
			Namespace:   qsec.GetNamespace(),
			Labels:      qsec.Spec.SecretLabels,
			Annotations: qsec.Spec.SecretAnnotations,
		},
		StringData: map[string]string{
			"private_key": string(key.PrivateKey),
			"public_key":  string(key.PublicKey),
		},
	}

	return r.createSecrets(ctx, qsec, secret)
}

func (r *ReconcileQuarksSecret) createSSHSecret(ctx context.Context, qsec *qsv1a1.QuarksSecret) error {
	key, err := r.generator.GenerateSSHKey(qsec.GetName())
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        qsec.Spec.SecretName,
			Namespace:   qsec.GetNamespace(),
			Labels:      qsec.Spec.SecretLabels,
			Annotations: qsec.Spec.SecretAnnotations,
		},
		StringData: map[string]string{
			"private_key":            string(key.PrivateKey),
			"public_key":             string(key.PublicKey),
			"public_key_fingerprint": key.Fingerprint,
		},
	}

	return r.createSecrets(ctx, qsec, secret)
}

func (r *ReconcileQuarksSecret) createBasicAuthSecret(ctx context.Context, qsec *qsv1a1.QuarksSecret) error {
	username := qsec.Spec.Request.BasicAuthRequest.Username
	if username == "" {
		username = r.generator.GeneratePassword(fmt.Sprintf("%s/username", qsec.Name), credsgen.PasswordGenerationRequest{})
	}
	password := r.generator.GeneratePassword(fmt.Sprintf("%s/password", qsec.Name), credsgen.PasswordGenerationRequest{})

	secret := &corev1.Secret{
		Type: corev1.SecretTypeBasicAuth,
		ObjectMeta: metav1.ObjectMeta{
			Name:      qsec.Spec.SecretName,
			Namespace: qsec.GetNamespace(),
		},
		StringData: map[string]string{
			"username": username,
			"password": password,
		},
	}

	return r.createSecrets(ctx, qsec, secret)
}

func (r *ReconcileQuarksSecret) createDockerConfigJSON(ctx context.Context, qsec *qsv1a1.QuarksSecret) error {
	// Fetch username and password.
	username := ""
	if len(qsec.Spec.Request.ImageCredentialsRequest.Username.Name) > 0 {
		userSecret := &corev1.Secret{}
		userNamespacedName := types.NamespacedName{
			Namespace: qsec.Namespace,
			Name:      qsec.Spec.Request.ImageCredentialsRequest.Username.Name,
		}
		err := r.client.Get(ctx, userNamespacedName, userSecret)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return newSecNotReadyError("username secret not found")
			}
			return errors.Wrap(err, "getting username secret")
		}
		data, ok := userSecret.Data[qsec.Spec.Request.ImageCredentialsRequest.Username.Key]
		if !ok {
			return errors.Errorf("Failed to get username data by key: %s", qsec.Spec.Request.ImageCredentialsRequest.Username.Key)
		}
		username = string(data)
	}
	if username == "" {
		username = r.generator.GeneratePassword(fmt.Sprintf("%s/username", qsec.Name), credsgen.PasswordGenerationRequest{})
	}

	password := ""
	if len(qsec.Spec.Request.ImageCredentialsRequest.Password.Name) > 0 {
		passSecret := &corev1.Secret{}
		passNamespacedName := types.NamespacedName{
			Namespace: qsec.Namespace,
			Name:      qsec.Spec.Request.ImageCredentialsRequest.Password.Name,
		}
		err := r.client.Get(ctx, passNamespacedName, passSecret)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return newSecNotReadyError("password secret not found")
			}
			return errors.Wrap(err, "getting password secret")
		}
		data, ok := passSecret.Data[qsec.Spec.Request.ImageCredentialsRequest.Password.Key]
		if !ok {
			return errors.Errorf("Failed to get password data key: %s", qsec.Spec.Request.ImageCredentialsRequest.Password.Key)
		}
		password = string(data)
	}
	if password == "" {
		password = r.generator.GeneratePassword(fmt.Sprintf("%s/password", qsec.Name), credsgen.PasswordGenerationRequest{})
	}

	authEncode := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	dockerConfigJSONData := fmt.Sprintf("{\"auths\":{\"%s\":{\"username\":\"%s\",\"password\":\"%s\",\"email\":\"%s\",\"auth\":\"%s\"}}}",
		qsec.Spec.Request.ImageCredentialsRequest.Registry,
		username,
		password,
		qsec.Spec.Request.ImageCredentialsRequest.Email,
		authEncode,
	)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      qsec.Spec.SecretName,
			Namespace: qsec.GetNamespace(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		StringData: map[string]string{
			corev1.DockerConfigJsonKey: dockerConfigJSONData,
		},
	}

	return r.createSecrets(ctx, qsec, secret)
}
