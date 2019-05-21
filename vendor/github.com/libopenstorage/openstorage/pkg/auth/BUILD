load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "auth.go",
        "claims.go",
        "duration.go",
        "oidc.go",
        "selfsigned.go",
        "signature.go",
        "token.go",
        "tokengenerator.go",
        "userinfo.go",
    ],
    importmap = "k8s.io/kubernetes/vendor/github.com/libopenstorage/openstorage/pkg/auth",
    importpath = "github.com/libopenstorage/openstorage/pkg/auth",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/coreos/go-oidc:go_default_library",
        "//vendor/github.com/dgrijalva/jwt-go:go_default_library",
    ],
)

filegroup(
    name = "package-srcs",
    srcs = glob(["**"]),
    tags = ["automanaged"],
    visibility = ["//visibility:private"],
)

filegroup(
    name = "all-srcs",
    srcs = [":package-srcs"],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
