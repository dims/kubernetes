load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "backoff.go",
        "context.go",
        "exponential.go",
        "retry.go",
        "ticker.go",
        "tries.go",
    ],
    importmap = "k8s.io/kubernetes/vendor/github.com/cenkalti/backoff",
    importpath = "github.com/cenkalti/backoff",
    visibility = ["//visibility:public"],
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
