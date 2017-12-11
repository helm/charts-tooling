# Charts Tooling

This repository contains one time tooling to support Helm and the community
[charts](https://github.com/kubernetes/charts) repository. Regular tools or CI
should typically be part of that repository.

## Prerequisites

This repository uses dep to manage dependencies. Please install dep and run
`dep ensure` prior to using the tools.

## Current Tooling

### gen-owners

This tool can take a path to a chart file along with some other options,
generate an OWNERS file (even looking up names on GitHub), and (using options)
modify other parts of the chart to work with owners files.
