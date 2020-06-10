package component

import (
	"context"
	"crypto/rand"

	"github.com/hashicorp/go-version"
	pxutil "github.com/libopenstorage/operator/drivers/storage/portworx/util"
	corev1alpha1 "github.com/libopenstorage/operator/pkg/apis/core/v1alpha1"
	k8sutil "github.com/libopenstorage/operator/pkg/util/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// SecurityComponentName is the name for registering this component
	SecurityComponentName = "Security"
	// SecurityPXAdminSecretName is the admin secret name for PX security
	SecurityPXAdminSecretName = "px-admin"
	// SecurityPXSystemSecretName is the system secret name for PX security
	SecurityPXSystemSecretName = "px-system"
	// SecurityPXAuthSecretKey is the data key for any secret containing an auth secret
	SecurityPXAuthSecretKey = "auth-secret"
	// SecurityEnvKeyPortworxAuthSystemKey is the environment variable name for the PX security secret
	SecurityEnvKeyPortworxAuthSystemKey = "PORTWORX_AUTH_SYSTEM_KEY"
	// SecurityEnvPortworxAuthJwtSharedSecret is an environment variable defining the PX Security JWT secret
	SecurityEnvPortworxAuthJwtSharedSecret = "PORTWORX_AUTH_JWT_SHAREDSECRET"
	// SecurityEnvPortworxAuthJwtIssuer is an environment variable defining the PX Security JWT Issuer
	SecurityEnvPortworxAuthJwtIssuer = "PORTWORX_AUTH_JWT_ISSUER"
)

type security struct {
	k8sClient client.Client
}

// Initialize initializes the componenet
func (c *security) Initialize(
	k8sClient client.Client,
	k8sVersion version.Version,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
) {
	c.k8sClient = k8sClient
}

// IsEnabled checks if the components needs to be enabled based on the StorageCluster
func (c *security) IsEnabled(cluster *corev1alpha1.StorageCluster) bool {
	return cluster.Spec.Security != nil && *cluster.Spec.Security.Enabled
}

// Reconcile reconciles the component to match the current state of the StorageCluster
func (c *security) Reconcile(cluster *corev1alpha1.StorageCluster) error {
	ownerRef := metav1.NewControllerRef(cluster, pxutil.StorageClusterKind())

	err := c.setJwtIssuer(cluster, ownerRef)
	if err != nil {
		return err
	}

	err = c.createSystemSecret(cluster, ownerRef)
	if err != nil {
		return err
	}

	err = c.createAdminSecret(cluster, ownerRef)
	if err != nil {
		return err
	}

	return nil
}

// Delete deletes the component if present
func (c *security) Delete(cluster *corev1alpha1.StorageCluster) error {
	err := c.deleteSystemSecret(cluster)
	if err != nil {
		return err
	}

	err = c.deleteAdminSecret(cluster)
	if err != nil {
		return err
	}

	return nil
}

// MarkDeleted marks the component as deleted in situations like StorageCluster deletion
func (c *security) MarkDeleted() {

}

func (c *security) setJwtIssuer(
	cluster *corev1alpha1.StorageCluster,
	ownerRef *metav1.OwnerReference,
) error {
	for i := range cluster.Spec.Env {
		if cluster.Spec.Env[i].Name == SecurityEnvPortworxAuthJwtIssuer {
			value, err := pxutil.GetValueFromEnv(context.TODO(), c.k8sClient, &cluster.Spec.Env[i], cluster.Namespace)
			if err != nil {
				return err
			}

			// If value does not equal spec value, update to match the spec
			if value != *cluster.Spec.Security.Auth.Authenticators.SelfSigned.Issuer {
				cluster.Spec.Env[i].Value = *cluster.Spec.Security.Auth.Authenticators.SelfSigned.Issuer
				cluster.Spec.Env[i].ValueFrom = nil
			}

			// found and updated, exit
			return nil
		}
	}

	// if not found, add default jwt issuer
	cluster.Spec.Env = append(cluster.Spec.Env, v1.EnvVar{
		Name:  SecurityEnvPortworxAuthJwtIssuer,
		Value: *cluster.Spec.Security.Auth.Authenticators.SelfSigned.Issuer,
	})

	return nil
}

func (c *security) createAdminSecret(
	cluster *corev1alpha1.StorageCluster,
	ownerRef *metav1.OwnerReference,
) error {
	err := c.createAuthSecret(cluster, ownerRef, SecurityEnvPortworxAuthJwtSharedSecret, SecurityPXAdminSecretName)
	if err != nil {
		return err
	}

	return nil
}

func (c *security) createSystemSecret(
	cluster *corev1alpha1.StorageCluster,
	ownerRef *metav1.OwnerReference,
) error {
	err := c.createAuthSecret(cluster, ownerRef, SecurityEnvKeyPortworxAuthSystemKey, SecurityPXSystemSecretName)
	if err != nil {
		return err
	}

	return nil
}

func (c *security) createAuthSecret(
	cluster *corev1alpha1.StorageCluster,
	ownerRef *metav1.OwnerReference,
	envVarName string,
	secretName string,
) error {
	var authSecret string
	var err error

	for _, envVar := range cluster.Spec.Env {
		if envVar.Name == envVarName {
			authSecret, err = pxutil.GetValueFromEnv(context.TODO(), c.k8sClient, &envVar, cluster.Namespace)
			if err != nil {
				return err
			}
		}
	}

	// If we did not find a secret, generate and add it
	if authSecret == "" {
		authSecret, err = generateAuthSecret()
		if err != nil {
			return err
		}

		adminSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: cluster.ObjectMeta.Namespace,
			},
			StringData: map[string]string{
				SecurityPXAuthSecretKey: authSecret,
			},
		}

		err = k8sutil.CreateOrUpdateSecret(c.k8sClient, adminSecret, ownerRef)
		if err != nil {
			return err
		}

		err = addAdminSecretEnv(cluster, envVarName, secretName)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateAuthSecret() (string, error) {
	var password = make([]byte, 32)
	_, err := rand.Read(password)
	if err != nil {
		return "", err
	}

	return string(password), nil
}

func addAdminSecretEnv(cluster *corev1alpha1.StorageCluster, envVar string, secretName string) error {
	// set as generated secret as environment variable
	if len(cluster.Spec.Env) == 0 {
		cluster.Spec.Env = make([]v1.EnvVar, 0)
	}
	var present bool
	for i := range cluster.Spec.Env {
		if cluster.Spec.Env[i].Name == envVar {
			present = true
		}
	}
	if !present {
		cluster.Spec.Env = append(cluster.Spec.Env, v1.EnvVar{
			Name: envVar,
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: secretName,
					},
					Key: SecurityPXAuthSecretKey,
				},
			},
		})
	}

	return nil
}

func (c *security) deleteSystemSecret(cluster *corev1alpha1.StorageCluster) error {
	systemSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecurityPXSystemSecretName,
			Namespace: cluster.ObjectMeta.Namespace,
		},
	}
	return c.k8sClient.Delete(context.TODO(), systemSecret)
}

func (c *security) deleteAdminSecret(cluster *corev1alpha1.StorageCluster) error {
	adminSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecurityPXAdminSecretName,
			Namespace: cluster.ObjectMeta.Namespace,
		},
	}
	return c.k8sClient.Delete(context.TODO(), adminSecret)
}

// RegisterSecurityComponent registers the security component
func RegisterSecurityComponent() {
	Register(SecurityComponentName, &security{})
}

func init() {
	RegisterSecurityComponent()
}
