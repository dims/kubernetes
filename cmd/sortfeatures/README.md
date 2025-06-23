# sortfeatures

`sortfeatures` is a command-line tool that sorts feature declarations in Kubernetes feature files alphabetically.

## Purpose

In Kubernetes, feature gates should be listed in alphabetical, case-sensitive (upper before any lower case character) order to reduce the risk of code conflicts and improve readability. This tool enforces this convention by automatically sorting feature declarations in specified files.

## Usage

```bash
sortfeatures [flags] [files...]
```

### Flags

- `--files`: One or more file paths to process
- `--force`, `-f`: Force update even if the file is already sorted

### Examples

Process specific files:
```bash
sortfeatures --files pkg/features/kube_features.go staging/src/k8s.io/apiserver/pkg/features/kube_features.go
```

Process files using positional arguments:
```bash
sortfeatures pkg/features/kube_features.go staging/src/k8s.io/apiserver/pkg/features/kube_features.go
```

Force update even if files are already sorted:
```bash
sortfeatures --force pkg/features/kube_features.go
```

## How It Works

The tool performs the following steps for each specified file:

1. Parses the Go source file
2. Identifies var/const blocks containing feature declarations
3. Extracts features with their associated comments
4. Sorts the features alphabetically by name
5. Updates the file if the order has changed or if `--force` is specified

## Integration with hack/update-sortfeatures.sh

This tool is used by the `hack/update-sortfeatures.sh` script, which is the recommended way to sort feature declarations in the Kubernetes codebase. The script automatically processes the standard set of feature files:

```bash
hack/update-sortfeatures.sh
```

You can also specify particular files to process:

```bash
hack/update-sortfeatures.sh path/to/file1.go path/to/file2.go
```

## Files Typically Processed

The standard set of files processed by `hack/update-sortfeatures.sh` includes:

- `pkg/features/kube_features.go`
- `staging/src/k8s.io/apiserver/pkg/features/kube_features.go`
- `staging/src/k8s.io/client-go/features/known_features.go`
- `staging/src/k8s.io/controller-manager/pkg/features/kube_features.go`
- `staging/src/k8s.io/apiextensions-apiserver/pkg/features/kube_features.go`
- `test/e2e/feature/feature.go`
- `test/e2e/environment/environment.go`

## Related Tools

- **sortedfeatures linter**: A golangci-lint plugin that checks if feature gates are sorted alphabetically. When it detects unsorted features, it suggests running `hack/update-sortfeatures.sh` to fix the issues. See [hack/tools/golangci-lint/sortedfeatures](../../hack/tools/golangci-lint/sortedfeatures) for more information.

## Development

If you need to modify this tool, ensure that any changes are also reflected in the corresponding linter to maintain consistency in how features are sorted across the Kubernetes codebase.
