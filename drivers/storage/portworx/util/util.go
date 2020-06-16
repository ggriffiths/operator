package util

import (
	"context"
	"crypto/x509"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/libopenstorage/openstorage/pkg/auth"
	"github.com/libopenstorage/openstorage/pkg/grpcserver"
	corev1alpha1 "github.com/libopenstorage/operator/pkg/apis/core/v1alpha1"
	"github.com/libopenstorage/operator/pkg/controller/storagecluster"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DriverName name of the portworx driver
	DriverName = "portworx"
	// DefaultStartPort is the default start port for Portworx
	DefaultStartPort = 9001
	// DefaultOpenshiftStartPort is the default start port for Portworx on OpenShift
	DefaultOpenshiftStartPort = 17001
	// PortworxSpecsDir is the directory where all the Portworx specs are stored
	PortworxSpecsDir = "/configs"

	// PortworxServiceAccountName name of the Portworx service account
	PortworxServiceAccountName = "portworx"
	// PortworxServiceName name of the Portworx Kubernetes service
	PortworxServiceName = "portworx-service"
	// PortworxRESTPortName name of the Portworx API port
	PortworxRESTPortName = "px-api"
	// PortworxSDKPortName name of the Portworx SDK port
	PortworxSDKPortName = "px-sdk"
	// PortworxKVDBPortName name of the Portworx internal KVDB port
	PortworxKVDBPortName = "px-kvdb"

	// AnnotationIsPKS annotation indicating whether it is a PKS cluster
	AnnotationIsPKS = pxAnnotationPrefix + "/is-pks"
	// AnnotationIsGKE annotation indicating whether it is a GKE cluster
	AnnotationIsGKE = pxAnnotationPrefix + "/is-gke"
	// AnnotationIsAKS annotation indicating whether it is an AKS cluster
	AnnotationIsAKS = pxAnnotationPrefix + "/is-aks"
	// AnnotationIsEKS annotation indicating whether it is an EKS cluster
	AnnotationIsEKS = pxAnnotationPrefix + "/is-eks"
	// AnnotationIsOpenshift annotation indicating whether it is an OpenShift cluster
	AnnotationIsOpenshift = pxAnnotationPrefix + "/is-openshift"
	// AnnotationPVCController annotation indicating whether to deploy a PVC controller
	AnnotationPVCController = pxAnnotationPrefix + "/pvc-controller"
	// AnnotationPVCControllerCPU annotation for overriding the default CPU for PVC
	// controller deployment
	AnnotationPVCControllerCPU = pxAnnotationPrefix + "/pvc-controller-cpu"
	// AnnotationAutopilotCPU annotation for overriding the default CPU for Autopilot
	AnnotationAutopilotCPU = pxAnnotationPrefix + "/autopilot-cpu"
	// AnnotationServiceType annotation indicating k8s service type for all services
	// deployed by the operator
	AnnotationServiceType = pxAnnotationPrefix + "/service-type"
	// AnnotationPXVersion annotation indicating the portworx semantic version
	AnnotationPXVersion = pxAnnotationPrefix + "/px-version"
	// AnnotationDisableStorageClass annotation to disable installing default portworx
	// storage classes
	AnnotationDisableStorageClass = pxAnnotationPrefix + "/disable-storage-class"

	// EnvKeyPXImage key for the environment variable that specifies Portworx image
	EnvKeyPXImage = "PX_IMAGE"
	// EnvKeyPortworxNamespace key for the env var which tells namespace in which
	// Portworx is installed
	EnvKeyPortworxNamespace = "PX_NAMESPACE"
	// EnvKeyPortworxServiceName key for the env var which tells the name of the
	// portworx service to be used
	EnvKeyPortworxServiceName = "PX_SERVICE_NAME"
	// EnvKeyPortworxSecretsNamespace key for the env var which tells the namespace
	// where portworx should look for secrets
	EnvKeyPortworxSecretsNamespace = "PX_SECRETS_NAMESPACE"
	// EnvKeyDeprecatedCSIDriverName key for the env var that can force Portworx
	// to use the deprecated CSI driver name
	EnvKeyDeprecatedCSIDriverName = "PORTWORX_USEDEPRECATED_CSIDRIVERNAME"
	// EnvKeyDisableCSIAlpha key for the env var that is used to disable CSI
	// alpha features
	EnvKeyDisableCSIAlpha = "PORTWORX_DISABLE_CSI_ALPHA"
	// EnvKeyPortworxEnableTLS is a flag for enabling operator TLS with PX
	EnvKeyPortworxEnableTLS = "PX_ENABLE_TLS"
	// EnvKeyPortworxAuthSystemKey is the environment variable name for the PX security secret
	EnvKeyPortworxAuthSystemKey = "PORTWORX_AUTH_SYSTEM_KEY"
	// EnvKeyPortworxAuthJwtSharedSecret is an environment variable defining the PX Security JWT secret
	EnvKeyPortworxAuthJwtSharedSecret = "PORTWORX_AUTH_JWT_SHAREDSECRET"
	// EnvKeyPortworxAuthJwtIssuer is an environment variable defining the PX Security JWT Issuer
	EnvKeyPortworxAuthJwtIssuer = "PORTWORX_AUTH_JWT_ISSUER"

	pxAnnotationPrefix = "portworx.io"
	labelKeyName       = "name"
	defaultSDKPort     = 9020
)

var (
	// SpecsBaseDir functions returns the base directory for specs. This is extracted as
	// variable for testing. DO NOT change the value of the function unless for testing.
	SpecsBaseDir = getSpecsBaseDir
)

// IsPortworxEnabled returns true if portworx is not explicitly disabled using the annotation
func IsPortworxEnabled(cluster *corev1alpha1.StorageCluster) bool {
	disabled, err := strconv.ParseBool(cluster.Annotations[storagecluster.AnnotationDisableStorage])
	return err != nil || !disabled
}

// IsPKS returns true if the annotation has a PKS annotation and is true value
func IsPKS(cluster *corev1alpha1.StorageCluster) bool {
	enabled, err := strconv.ParseBool(cluster.Annotations[AnnotationIsPKS])
	return err == nil && enabled
}

// IsGKE returns true if the annotation has a GKE annotation and is true value
func IsGKE(cluster *corev1alpha1.StorageCluster) bool {
	enabled, err := strconv.ParseBool(cluster.Annotations[AnnotationIsGKE])
	return err == nil && enabled
}

// IsAKS returns true if the annotation has an AKS annotation and is true value
func IsAKS(cluster *corev1alpha1.StorageCluster) bool {
	enabled, err := strconv.ParseBool(cluster.Annotations[AnnotationIsAKS])
	return err == nil && enabled
}

// IsEKS returns true if the annotation has an EKS annotation and is true value
func IsEKS(cluster *corev1alpha1.StorageCluster) bool {
	enabled, err := strconv.ParseBool(cluster.Annotations[AnnotationIsEKS])
	return err == nil && enabled
}

// IsOpenshift returns true if the annotation has an OpenShift annotation and is true value
func IsOpenshift(cluster *corev1alpha1.StorageCluster) bool {
	enabled, err := strconv.ParseBool(cluster.Annotations[AnnotationIsOpenshift])
	return err == nil && enabled
}

// StorageClassEnabled returns true if default portworx storage classes are disabled
func StorageClassEnabled(cluster *corev1alpha1.StorageCluster) bool {
	disabled, err := strconv.ParseBool(cluster.Annotations[AnnotationDisableStorageClass])
	return err != nil || !disabled
}

// ServiceType returns the k8s service type from cluster annotations if present
func ServiceType(cluster *corev1alpha1.StorageCluster) v1.ServiceType {
	var serviceType v1.ServiceType
	if val, exists := cluster.Annotations[AnnotationServiceType]; exists {
		st := v1.ServiceType(val)
		if st == v1.ServiceTypeClusterIP ||
			st == v1.ServiceTypeNodePort ||
			st == v1.ServiceTypeLoadBalancer {
			serviceType = st
		}
	}
	return serviceType
}

// ImagePullPolicy returns the image pull policy from the cluster spec if present,
// else returns v1.PullAlways
func ImagePullPolicy(cluster *corev1alpha1.StorageCluster) v1.PullPolicy {
	imagePullPolicy := v1.PullAlways
	if cluster.Spec.ImagePullPolicy == v1.PullNever ||
		cluster.Spec.ImagePullPolicy == v1.PullIfNotPresent {
		imagePullPolicy = cluster.Spec.ImagePullPolicy
	}
	return imagePullPolicy
}

// StartPort returns the start from the cluster if present,
// else return the default start port
func StartPort(cluster *corev1alpha1.StorageCluster) int {
	startPort := DefaultStartPort
	if cluster.Spec.StartPort != nil {
		startPort = int(*cluster.Spec.StartPort)
	} else if IsOpenshift(cluster) {
		startPort = DefaultOpenshiftStartPort
	}
	return startPort
}

// UseDeprecatedCSIDriverName returns true if the cluster env variables has
// an override, else returns false.
func UseDeprecatedCSIDriverName(cluster *corev1alpha1.StorageCluster) bool {
	for _, env := range cluster.Spec.Env {
		if env.Name == EnvKeyDeprecatedCSIDriverName {
			value, err := strconv.ParseBool(env.Value)
			return err == nil && value
		}
	}
	return false
}

// DisableCSIAlpha returns true if the cluster env variables has a variable to disable
// CSI alpha features, else returns false.
func DisableCSIAlpha(cluster *corev1alpha1.StorageCluster) bool {
	for _, env := range cluster.Spec.Env {
		if env.Name == EnvKeyDisableCSIAlpha {
			value, err := strconv.ParseBool(env.Value)
			return err == nil && value
		}
	}
	return false
}

// GetPortworxVersion returns the Portworx version based on the image provided.
// We first look at spec.Image, if not valid image tag found, we check the PX_IMAGE
// env variable. If that is not present or invalid semvar, then we fallback to an
// annotation portworx.io/px-version; else we return int max as the version.
func GetPortworxVersion(cluster *corev1alpha1.StorageCluster) *version.Version {
	var (
		err       error
		pxVersion *version.Version
	)

	pxImage := cluster.Spec.Image
	for _, env := range cluster.Spec.Env {
		if env.Name == EnvKeyPXImage {
			pxImage = env.Value
			break
		}
	}

	parts := strings.Split(pxImage, ":")
	if len(parts) >= 2 {
		pxVersionStr := parts[len(parts)-1]
		pxVersion, err = version.NewSemver(pxVersionStr)
		if err != nil {
			logrus.Warnf("Invalid PX version %s extracted from image name: %v", pxVersionStr, err)
			if pxVersionStr, exists := cluster.Annotations[AnnotationPXVersion]; exists {
				pxVersion, err = version.NewSemver(pxVersionStr)
				if err != nil {
					logrus.Warnf("Invalid PX version %s extracted from annotation: %v", pxVersionStr, err)
				}
			}
		}
	}

	if pxVersion == nil {
		pxVersion, _ = version.NewVersion(strconv.FormatInt(math.MaxInt64, 10))
	}
	return pxVersion
}

// GetImageTag returns the tag of the image
func GetImageTag(image string) string {
	if parts := strings.Split(image, ":"); len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}

// SelectorLabels returns the labels that are used to select Portworx pods
func SelectorLabels() map[string]string {
	return map[string]string{
		labelKeyName: DriverName,
	}
}

// StorageClusterKind returns the GroupVersionKind for StorageCluster
func StorageClusterKind() schema.GroupVersionKind {
	return corev1alpha1.SchemeGroupVersion.WithKind("StorageCluster")
}

// GetClusterEnvVarValue returns the environment variable value for a cluster.
// Note: This does not
func GetClusterEnvVarValue(ctx context.Context, cluster *corev1alpha1.StorageCluster, envKey string) string {
	for _, envVar := range cluster.Spec.Env {
		if envVar.Name == envKey {
			return envVar.Value
		}
	}

	return ""
}

// GetValueFromEnvVar returns the value of v1.EnvVar Value or ValueFrom
func GetValueFromEnvVar(ctx context.Context, client client.Client, envVar *v1.EnvVar, namespace string) (string, error) {
	if valueFrom := envVar.ValueFrom; valueFrom != nil {
		if valueFrom.SecretKeyRef != nil {
			key := valueFrom.SecretKeyRef.Key
			secretName := valueFrom.SecretKeyRef.Name

			// Get secret key
			secret := &v1.Secret{}
			err := client.Get(ctx, types.NamespacedName{
				Name:      secretName,
				Namespace: namespace,
			}, secret)
			if err != nil {
				return "", err
			}
			value := secret.Data[key]
			if len(value) == 0 {
				return "", fmt.Errorf("failed to find env var value %s in secret %s in namespace %s", key, secretName, namespace)
			}

			return string(value), nil
		} else if valueFrom.ConfigMapKeyRef != nil {
			cmName := valueFrom.ConfigMapKeyRef.Name
			key := valueFrom.ConfigMapKeyRef.Key
			configMap := &v1.ConfigMap{}
			if err := client.Get(ctx, types.NamespacedName{
				Name:      cmName,
				Namespace: namespace,
			}, configMap); err != nil {
				return "", err
			}

			value, ok := configMap.Data[key]
			if !ok {
				return "", fmt.Errorf("failed to find env var value %s in configmap %s in namespace %s", key, cmName, namespace)
			}

			return value, nil
		}
	} else {
		return envVar.Value, nil
	}

	return "", nil
}

func getSpecsBaseDir() string {
	return PortworxSpecsDir
}

// GetPortworxConn returns a new Portworx SDK client
func GetPortworxConn(sdkConn *grpc.ClientConn, k8sClient client.Client, namespace string) (*grpc.ClientConn, error) {
	if sdkConn != nil {
		return sdkConn, nil
	}

	pxService := &v1.Service{}
	err := k8sClient.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      PortworxServiceName,
			Namespace: namespace,
		},
		pxService,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s service spec: %v", err)
	} else if len(pxService.Spec.ClusterIP) == 0 {
		return nil, fmt.Errorf("failed to get endpoint for portworx volume driver")
	}

	endpoint := pxService.Spec.ClusterIP
	sdkPort := defaultSDKPort

	// Get the ports from service
	for _, pxServicePort := range pxService.Spec.Ports {
		if pxServicePort.Name == PortworxSDKPortName && pxServicePort.Port != 0 {
			sdkPort = int(pxServicePort.Port)
		}
	}

	endpoint = fmt.Sprintf("%s:%d", endpoint, sdkPort)
	return GetGrpcConn(endpoint)
}

// GetGrpcConn creates a new gRPC connection to a given endpoint
func GetGrpcConn(endpoint string) (*grpc.ClientConn, error) {
	dialOptions, err := GetDialOptions(IsTLSEnabled())
	if err != nil {
		return nil, err
	}
	sdkConn, err := grpcserver.Connect(endpoint, dialOptions)
	if err != nil {
		return nil, fmt.Errorf("error connecting to GRPC server [%s]: %v", endpoint, err)
	}
	return sdkConn, nil
}

// GetDialOptions is a gRPC utility to get dial options for a connection
func GetDialOptions(tls bool) ([]grpc.DialOption, error) {
	if !tls {
		return []grpc.DialOption{grpc.WithInsecure()}, nil
	}
	capool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load CA system certs: %v", err)
	}
	return []grpc.DialOption{grpc.WithTransportCredentials(
		credentials.NewClientTLSFromCert(capool, ""),
	)}, nil
}

// IsTLSEnabled checks if TLS is enabled for the operator
func IsTLSEnabled() bool {
	enabled, err := strconv.ParseBool(os.Getenv(EnvKeyPortworxEnableTLS))
	return err == nil && enabled
}

// GetOperatorToken generates an auth token given a secret key
func GetOperatorToken(
	cluster *corev1alpha1.StorageCluster,
	secretkey string,
) (string, error) {
	claims := &auth.Claims{
		Issuer:  *cluster.Spec.Security.Auth.Authenticators.SelfSigned.Issuer,
		Subject: "operator@portworx.io",
		Name:    "operator communications",
		Email:   "operator@portworx.io",
		Roles:   []string{"system.admin"},
		Groups:  []string{"*"},
	}

	signature, err := auth.NewSignatureSharedSecret(string(secretkey))
	if err != nil {
		return "", err
	}
	token, err := auth.Token(claims, signature, &auth.Options{
		Expiration: time.Now().
			Add(cluster.Spec.Security.Auth.Authenticators.SelfSigned.TokenLifetime.Duration).Unix(),
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

// GetAdminSecret gets the admin secret from a pre-configured environment variable
func GetAdminSecret(
	ctx context.Context,
	cluster *corev1alpha1.StorageCluster,
	k8sClient client.Client,
) (string, error) {
	var authSecret string
	var err error

	// check for provided secret
	for _, envVar := range cluster.Spec.Env {
		if envVar.Name == EnvKeyPortworxAuthJwtSharedSecret {
			authSecret, err = GetValueFromEnvVar(ctx, k8sClient, &envVar, cluster.Namespace)
			if err != nil {
				return "", err
			}

			return authSecret, nil
		}
	}

	return "", nil
}

// SecurityEnabled checks if the security flag is set for a cluster
func SecurityEnabled(cluster *corev1alpha1.StorageCluster) bool {
	return cluster.Spec.Security != nil && cluster.Spec.Security.Enabled
}

// SetupContextWithToken Gets token or from secret for authenticating with the SDK server
func SetupContextWithToken(ctx context.Context, cluster *corev1alpha1.StorageCluster, k8sClient client.Client) (context.Context, error) {
	// auth not declared in cluster spec
	if !SecurityEnabled(cluster) {
		return ctx, nil
	}

	pxAuthSecret, err := GetAdminSecret(ctx, cluster, k8sClient)
	if err != nil {
		return ctx, fmt.Errorf("failed to get auth secret: %v", err.Error())
	}
	if pxAuthSecret == "" {
		return ctx, nil
	}

	// Generate token and add to metadata
	token, err := GetOperatorToken(cluster, string(pxAuthSecret))
	if err != nil {
		return ctx, fmt.Errorf("failed to create operator token: %v", err.Error())
	}
	md := metadata.New(map[string]string{
		"authorization": "bearer " + token,
	})
	return metadata.NewOutgoingContext(ctx, md), nil
}
