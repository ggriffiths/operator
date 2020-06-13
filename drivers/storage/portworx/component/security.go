package component

import (
	"context"
	"crypto/rand"

	"github.com/hashicorp/go-version"
	"github.com/libopenstorage/openstorage/api"
	pxutil "github.com/libopenstorage/operator/drivers/storage/portworx/util"
	corev1alpha1 "github.com/libopenstorage/operator/pkg/apis/core/v1alpha1"
	k8sutil "github.com/libopenstorage/operator/pkg/util/k8s"
	"google.golang.org/grpc"
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
	// SecuritySystemGuestRoleName is the role name to maintain for the guest role
	SecuritySystemGuestRoleName = "system.guest"
)

var guestRoleEnabled = api.SdkRole{
	Name: SecuritySystemGuestRoleName,
	Rules: []*api.SdkRule{
		{
			Services: []string{"mountattach", "volume", "cloudbackup", "migrate"},
			Apis:     []string{"*"},
		},
		{
			Services: []string{"identity"},
			Apis:     []string{"version"},
		},
	},
}

var guestRoleDisabled = api.SdkRole{
	Name: SecuritySystemGuestRoleName,
	Rules: []*api.SdkRule{
		{
			Services: []string{"!*"},
			Apis:     []string{"!*"},
		},
	},
}

type security struct {
	k8sClient client.Client
	sdkConn   *grpc.ClientConn
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
	return pxutil.SecurityEnabled(cluster)
}

// Reconcile reconciles the component to match the current state of the StorageCluster
func (c *security) Reconcile(cluster *corev1alpha1.StorageCluster) error {
	ownerRef := metav1.NewControllerRef(cluster, pxutil.StorageClusterKind())

	err := c.updateSystemGuestRole(cluster)
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

func (c *security) createAdminSecret(
	cluster *corev1alpha1.StorageCluster,
	ownerRef *metav1.OwnerReference,
) error {
	err := c.createAuthSecret(cluster, ownerRef, pxutil.EnvKeyPortworxAuthJwtSharedSecret, SecurityPXAdminSecretName)
	if err != nil {
		return err
	}

	return nil
}

func (c *security) createSystemSecret(
	cluster *corev1alpha1.StorageCluster,
	ownerRef *metav1.OwnerReference,
) error {
	err := c.createAuthSecret(cluster, ownerRef, pxutil.EnvKeyPortworxAuthSystemKey, SecurityPXSystemSecretName)
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
			authSecret, err = pxutil.GetValueFromEnvVar(context.TODO(), c.k8sClient, &envVar, cluster.Namespace)
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

func (c *security) updateSystemGuestRole(cluster *corev1alpha1.StorageCluster) error {
	// managed, do not interfere with system.guest role
	if *cluster.Spec.Security.Auth.GuestAccess == corev1alpha1.GuestRoleManaged {
		return nil
	}

	var err error
	pxConn, err := pxutil.GetPortworxConn(c.sdkConn, c.k8sClient, cluster.Namespace)
	if err != nil {
		return err
	}

	roleClient := api.NewOpenStorageRoleClient(pxConn)
	ctx, err := pxutil.SetupContextWithToken(context.Background(), cluster, c.k8sClient)
	if err != nil {
		return err
	}

	var desiredRole *api.SdkRole

	if *cluster.Spec.Security.Auth.GuestAccess == corev1alpha1.GuestRoleEnabled {
		desiredRole = &guestRoleEnabled
	} else {
		desiredRole = &guestRoleDisabled
	}

	_, err = roleClient.Update(ctx, &api.SdkRoleUpdateRequest{
		Role: desiredRole,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *security) deleteSystemSecret(cluster *corev1alpha1.StorageCluster) error {
	ownerRef := metav1.NewControllerRef(cluster, pxutil.StorageClusterKind())
	return k8sutil.DeleteSecret(c.k8sClient, SecurityPXSystemSecretName, cluster.Namespace, *ownerRef)
}

func (c *security) deleteAdminSecret(cluster *corev1alpha1.StorageCluster) error {
	ownerRef := metav1.NewControllerRef(cluster, pxutil.StorageClusterKind())
	return k8sutil.DeleteSecret(c.k8sClient, SecurityPXAdminSecretName, cluster.Namespace, *ownerRef)
}

// RegisterSecurityComponent registers the security component
func RegisterSecurityComponent() {
	Register(SecurityComponentName, &security{})
}

func init() {
	RegisterSecurityComponent()
}
