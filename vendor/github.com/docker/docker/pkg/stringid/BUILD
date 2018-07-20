load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["stringid.go"],
    importmap = "k8s.io/kubernetes/vendor/github.com/docker/docker/pkg/stringid",
    importpath = "github.com/docker/docker/pkg/stringid",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["stringid_test.go"],
    embed = [":go_default_library"],
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
