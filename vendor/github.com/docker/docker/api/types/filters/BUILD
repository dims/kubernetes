load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["parse.go"],
    importmap = "k8s.io/kubernetes/vendor/github.com/docker/docker/api/types/filters",
    importpath = "github.com/docker/docker/api/types/filters",
    visibility = ["//visibility:public"],
    deps = ["//vendor/github.com/docker/docker/api/types/versions:go_default_library"],
)

go_test(
    name = "go_default_test",
    srcs = [
        "example_test.go",
        "parse_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//vendor/github.com/stretchr/testify/assert:go_default_library",
        "//vendor/github.com/stretchr/testify/require:go_default_library",
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
