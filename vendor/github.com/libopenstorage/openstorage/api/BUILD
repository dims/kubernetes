load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "api.go",
        "api.pb.go",
        "api.pb.gw.go",
        "group.go",
        "ownership.go",
        "status.go",
    ],
    importmap = "k8s.io/kubernetes/vendor/github.com/libopenstorage/openstorage/api",
    importpath = "github.com/libopenstorage/openstorage/api",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/timestamp:go_default_library",
        "//vendor/github.com/grpc-ecosystem/grpc-gateway/runtime:go_default_library",
        "//vendor/github.com/grpc-ecosystem/grpc-gateway/utilities:go_default_library",
        "//vendor/github.com/libopenstorage/openstorage/pkg/auth:go_default_library",
        "//vendor/github.com/mohae/deepcopy:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/api/annotations:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
        "//vendor/google.golang.org/grpc/codes:go_default_library",
        "//vendor/google.golang.org/grpc/grpclog:go_default_library",
        "//vendor/google.golang.org/grpc/status:go_default_library",
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
    srcs = [
        ":package-srcs",
        "//vendor/github.com/libopenstorage/openstorage/api/client:all-srcs",
        "//vendor/github.com/libopenstorage/openstorage/api/errors:all-srcs",
        "//vendor/github.com/libopenstorage/openstorage/api/spec:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
