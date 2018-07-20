load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "backend.go",
        "filter.go",
        "network.go",
        "network_routes.go",
    ],
    importmap = "k8s.io/kubernetes/vendor/github.com/docker/docker/api/server/router/network",
    importpath = "github.com/docker/docker/api/server/router/network",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/docker/docker/api/server/httputils:go_default_library",
        "//vendor/github.com/docker/docker/api/server/router:go_default_library",
        "//vendor/github.com/docker/docker/api/types:go_default_library",
        "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        "//vendor/github.com/docker/docker/api/types/network:go_default_library",
        "//vendor/github.com/docker/docker/api/types/versions:go_default_library",
        "//vendor/github.com/docker/docker/daemon/cluster:go_default_library",
        "//vendor/github.com/docker/docker/runconfig:go_default_library",
        "//vendor/github.com/docker/libnetwork:go_default_library",
        "//vendor/github.com/docker/libnetwork/datastore:go_default_library",
        "//vendor/github.com/docker/libnetwork/networkdb:go_default_library",
        "//vendor/github.com/pkg/errors:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["filter_test.go"],
    embed = [":go_default_library"],
    deps = select({
        "@io_bazel_rules_go//go/platform:android": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "@io_bazel_rules_go//go/platform:darwin": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "@io_bazel_rules_go//go/platform:dragonfly": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "@io_bazel_rules_go//go/platform:freebsd": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "@io_bazel_rules_go//go/platform:linux": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "@io_bazel_rules_go//go/platform:nacl": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "@io_bazel_rules_go//go/platform:netbsd": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "@io_bazel_rules_go//go/platform:openbsd": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "@io_bazel_rules_go//go/platform:plan9": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "@io_bazel_rules_go//go/platform:solaris": [
            "//vendor/github.com/docker/docker/api/types:go_default_library",
            "//vendor/github.com/docker/docker/api/types/filters:go_default_library",
        ],
        "//conditions:default": [],
    }),
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
