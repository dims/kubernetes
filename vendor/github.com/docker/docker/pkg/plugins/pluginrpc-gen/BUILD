load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
        "parser.go",
        "template.go",
    ],
    importmap = "k8s.io/kubernetes/vendor/github.com/docker/docker/pkg/plugins/pluginrpc-gen",
    importpath = "github.com/docker/docker/pkg/plugins/pluginrpc-gen",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "pluginrpc-gen",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["parser_test.go"],
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
    srcs = [
        ":package-srcs",
        "//vendor/github.com/docker/docker/pkg/plugins/pluginrpc-gen/fixtures:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
