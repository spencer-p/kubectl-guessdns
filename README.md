# kubectl-guessdns

A `kubectl` plugin that returns the most likely cluster DNS entry for a matching Service or Pod using fuzzy name matching (Levenshtein distance).

## Features

- **Fuzzy Matching**: Uses Levenshtein distance to find the best match even with partial or slightly different names.
- **Service Priority**: Always checks for Services first. If no likely candidate is found, it falls back to Pods.
- **Natural Querying**: Accepts multiple arguments which are joined by `-` to form the search query.
- **Case-Insensitive**: Matches names regardless of casing.

## How it works

1. Joins all command-line arguments with `-`.
2. Lists all **Services** and calculates similarity.
3. Returns the Service FQDN if a match exceeds the threshold (0.4).
4. If no Service matches, it repeats the process for **Pods**.
5. Returns the Pod DNS name (IP-based) for the best-matching Pod.

## Installation

### Prerequisites
- Go 1.25+
- Access to a Kubernetes cluster (via `kubeconfig`)

### Build
```bash
go build -o kubectl-guessdns main.go
```

### Use as a kubectl plugin
To use it as a plugin, ensure the binary is named `kubectl-guessdns` and is available in your `PATH`.

```bash
# Example: move to a local bin directory
mv kubectl-guessdns /usr/local/bin/

# Now you can use it via kubectl
kubectl guessdns <query>
```

## Examples

### Guessing a Service
```bash
$ kubectl guessdns gemma head
gemma-2b-it-head-svc.default.svc.cluster.local
```

### Guessing a Pod
```bash
$ kubectl guessdns gemma worker
10-80-7-5.default.pod.cluster.local
# Based on pod: gemma-2b-it-fsrdf-head-j47nr
```

## Configuration

The following constants in `main.go` can be tuned:
- `threshold`: Minimum similarity score (0.0 to 1.0) to consider a match. Default is `0.4`.
- `clusterDomain`: The cluster's base domain. Default is `cluster.local`.
