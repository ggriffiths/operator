module github.com/libopenstorage/operator

go 1.15

require (
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/coreos/go-oidc v2.2.1+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/golang/mock v1.4.4
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/hashicorp/consul v1.0.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-version v1.2.1
	github.com/libopenstorage/cloudops v0.0.0-20190815012442-6e0d676b6c3e
	github.com/libopenstorage/openstorage v8.0.1-0.20200730032300-bef55ec5700c+incompatible
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/operator-framework/operator-sdk v1.4.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/portworx/kvdb v0.0.0-20200723230726-2734b7f40194
	github.com/portworx/sched-ops v1.20.0-rc1
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/pquerna/ffjson v0.0.0-20180717144149-af8b230fcd20 // indirect
	github.com/prometheus-operator/prometheus-operator v0.46.0 // indirect
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.46.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/urfave/cli v1.22.1
	gomodules.xyz/jsonpatch v2.0.1+incompatible // indirect
	google.golang.org/grpc v1.33.2
	google.golang.org/grpc/examples v0.0.0-20210226164526-c949703b4b98 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/csi-translation-lib v0.0.0-20190913091657-9745ba0e69cf // indirect
	k8s.io/kubernetes v1.14.6
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/cluster-api v0.2.11
	sigs.k8s.io/controller-runtime v0.8.0
)

replace (
	github.com/coreos/prometheus-operator => github.com/prometheus-operator/prometheus-operator v0.46.0
	github.com/kubernetes-incubator/external-storage => github.com/libopenstorage/external-storage v5.1.1-0.20190919185747-9394ee8dd536+incompatible
	github.com/kubernetes-incubator/external-storage v0.0.0-00010101000000-000000000000 => github.com/libopenstorage/external-storage v5.1.1-0.20190919185747-9394ee8dd536+incompatible
	gopkg.in/fsnotify.v1 v1.4.7 => github.com/fsnotify/fsnotify v1.4.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.4
)
