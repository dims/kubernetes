// This is a generated file. Do not edit directly.

module k8s.io/kube-controller-manager

go 1.13

require (
	k8s.io/apimachinery v0.0.0
	k8s.io/component-base v0.0.0
)

replace (
	golang.org/x/exp => golang.org/x/exp v0.0.0-20190312203227-4b39c73a6495
	golang.org/x/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	golang.org/x/oauth2 => golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a // pinned to release-branch.go1.13
	golang.org/x/tools => golang.org/x/tools v0.0.0-20190821162956-65e3620a7ae7 // pinned to release-branch.go1.13
	k8s.io/api => ../api
	k8s.io/apimachinery => ../apimachinery
	k8s.io/client-go => ../client-go
	k8s.io/component-base => ../component-base
	k8s.io/kube-controller-manager => ../kube-controller-manager
)
