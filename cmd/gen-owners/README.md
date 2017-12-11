# gen-owners

This utility generates an OWNERS file for a chart

# Usage

`gen-owners` has 3 flags to pass in information.

* `-c [FILENAME]`: The path to the `Chart.yaml` file. Defaults to `Chart.yaml`.
  Can be a relative or absolute path.
* `-o`: If the created `OWNERS` file should be written to disk alongside the
  `Chart.yaml` file.
* `-i`: If the `OWNERS` file should be appended to the bottom of the `.helmignore`
  file.

Example usage:

```
$ gen-owners -c ~/Code/k8s/charts/stable/percona/Chart.yaml -o -i
Found github id "CaptTofu" for name "Patrick Galbraith"
OWNERS file content:
approvers:
- CaptTofu
reviewers:
- CaptTofu

Writing owners file
Appending OWNERS to .helmignore
```

## What gen-owners does not do

There are a couple important things this utility does not do.

1. The version is not incremented in the `Chart.yaml` file
1. Does not update names in the `Chart.yaml` file to the corresponting
   GitHub handle.

The reason these are not automated is the YAML output tooling restructures the
ordering in the `Chart.yaml` file so that keys are in alpha order rather than
keeping the current ordering.

PRs welcome to improve this.
