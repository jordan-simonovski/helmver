# Excluding charts from discovery

The `--exclude` flag filters out charts during discovery. It accepts glob patterns matched against each path component relative to the `--dir` root.

```bash
helmver check --dir helm/charts --exclude '[0-9]*'
helmver changeset --dir helm/charts --exclude '[0-9]*'
helmver apply --dir helm/charts --exclude '[0-9]*'
```

The flag is repeatable -- pass it multiple times to exclude several patterns:

```bash
helmver check --dir charts/ --exclude 'vendor' --exclude 'test-*'
```

Or as a comma-separated list:

```bash
helmver check --dir charts/ --exclude 'vendor,test-*'
```

## How matching works

Each pattern is tested against two things for every file and directory encountered during the recursive walk:

1. The **full relative path** from `--dir` (e.g. `ingress-nginx/4.13.3/ingress-nginx/Chart.yaml`)
2. The **base name** of each path component (e.g. `4.13.3`, `ingress-nginx`, `Chart.yaml`)

When a **directory** matches, the entire subtree is skipped -- nothing inside it is visited. This is both correct and efficient.

Patterns use Go's [`filepath.Match`](https://pkg.go.dev/path/filepath#Match) syntax:

| Pattern     | Matches                                           |
|-------------|---------------------------------------------------|
| `*`         | Any single path component                         |
| `?`         | Any single character                               |
| `[0-9]*`    | Components starting with a digit                   |
| `[a-z]??`   | Three-character lowercase components               |
| `vendor`    | Exact directory or file name `vendor`              |
| `test-*`    | Components starting with `test-`                   |

Patterns do **not** support `**` (recursive globbing) or `/` (path separators). Each pattern matches a single path component.

## Common use cases

### Vendored upstream charts with version directories

A layout like this stores upstream charts under version directories:

```
helm/charts/
  ingress-nginx/
    Chart.yaml                              # wrapper chart (you own this)
    4.13.3/ingress-nginx/Chart.yaml         # vendored upstream
    4.12.0/ingress-nginx/Chart.yaml         # vendored upstream
  cert-manager/
    Chart.yaml                              # wrapper chart
    1.14.0/cert-manager/Chart.yaml          # vendored upstream
```

Without `--exclude`, helmver discovers all five Chart.yaml files. The vendored ones share the same `name` field as the wrapper charts, causing collisions in `apply` and confusing output in `check`.

```bash
helmver check --dir helm/charts --exclude '[0-9]*'
```

This skips every directory whose name starts with a digit (`4.13.3`, `4.12.0`, `1.14.0`), discovering only the two wrapper charts.

### Skipping test or example charts

```bash
helmver check --dir charts/ --exclude 'test-*' --exclude 'examples'
```

### Skipping a specific chart by name

```bash
helmver check --dir charts/ --exclude 'legacy-api'
```

### Vendored dependencies under a vendor directory

```bash
helmver check --dir . --exclude 'vendor'
```

## Verifying what gets discovered

Run `helmver check` (without `--exclude`) first to see everything helmver finds, then add patterns until the output lists only charts you control.

```bash
# See all discovered charts
helmver check --dir helm/charts

# Add exclusion, verify the list shrinks
helmver check --dir helm/charts --exclude '[0-9]*'
```
