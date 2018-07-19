# Developers Guide

This guide explains how to set up your environment for developing on
Helm and Tiller.

## Prerequisites

- Go 1.6.0 or later
- Glide 0.10.2 or later
- kubectl 1.2 or later
- A Kubernetes cluster (optional)
- The gRPC toolchain

## Building Helm/Tiller

We use Make to build our programs. The simplest way to get started is:

```console
$ make boostrap build
```

This will build both Helm and Tiller. `make bootstrap` will attempt to
install certain tools if they are missing.

To run all of the tests (without running the tests for `vendor/`), run
`make test`.

To run Helm and Tiller locally, you can run `bin/helm` or `bin/tiller`.

- Helm and Tiller are known to run on Mac OSX and most Linuxes, including
  Alpine.
- Tiller must have access to a Kubernets cluster. It learns about the
  cluster by examining the Kube config files that `kubectl` uese.

## gRPC and Protobuf

Helm and Tiller communicate using gRPC. To get started with gRPC, you will need to...

- Install `protoc` for compiling protobuf files. Releases are
  [here](https://github.com/google/protobuf/releases)
- Run Helm's `make bootstrap` to generate the `protoc-gen-go` plugin and
  place it in `bin/`.

Note that you need to be on protobuf 3.x (`protoc --version`). The
version of `protoc-gen-go` is tied to the version of gRPC used in
Kubernetes. So the plugin is maintained locally.

While the gRPC and ProtoBuf specs remain silent on indentation, we
require that the indentation style matches the Go format specification.
Namely, protocol buffers should use tab-based indentation and rpc
declarations should follow the style of Go function declarations.

### The Helm API (HAPI)

We use gRPC as an API layer. See `pkg/proto/hapi` for the generated Go code,
and `_proto` for the protocol buffer definitions.

To regenerate the Go files from the protobuf source, `make protoc`.

## Docker Images

To build Docker images, use `make docker-build`.

Pre-build images are already available in the official Kubernetes Helm
GCR registry.

## Running a Local Cluster

You can run tests locally using the `scripts/local-cluster.sh` script to
start Kubernetes inside of a Docker container. For OS X, you will need
to be running `docker-machine`.

Tiller should run on any >= 1.2 Kubernetes cluster with beta extensions.

## Contribution Guidelines

We welcome contributions. This project has set up some guidelines in
order to ensure that (a) code quality remains high, (b) the project
remains consistent, and (c) contributions follow the open source legal
requirements. Our intent is not to burden contributors, but to build
elegant and high-quality open source code so that our users will benefit.

Make sure you have read and understood the main CONTRIBUTING guide:

https://github.com/kubernetes/helm/blob/master/CONTRIBUTING.md

### Git Conventions

We use Git for our version control system. The `master` branch is the
home of the current development candidate. Releases are tagged.

We accept changes to the code via GitHub Pull Requests (PRs). One
workflow for doing this is as follows:

1. Go to your `$GOPATH/k8s.io` directory and `git clone` the
   `github.com/kubernetes/helm` repository.
2. Fork that repository into your GitHub account
3. Add your repository as a remote for `$GOPATH/k8s.io/helm`
4. Create a new working branch (`git checkout -b feat/my-feature`) and
   do your work on that branch.
5. When you are ready for us to review, push your branch to GitHub, and
   then open a new pull request with us.

For Git commit messages, we follow the [Semantic Commit Messages](http://karma-runner.github.io/0.13/dev/git-commit-msg.html):

```
fix(helm): add --foo flag to 'helm install'

When 'helm install --foo bar' is run, this will print "foo" in the
output regardless of the outcome of the installation.

Closes #1234
```

Common commit types:

- fix: Fix a bug or error
- feat: Add a new feature
- docs: Change documentation
- test: Improve testing

Common scopes:

- helm: The Helm CLI
- tiller: The Tiller server
- proto: Protobuf definitions
- pkg/lint: The lint package. Follow a similar convention for any
  package,
- `*`: two or more scopes

Read more:
- The [Deis Guidelines](https://github.com/deis/workflow/blob/master/src/contributing/submitting-a-pull-request.md)
  were the inspiration for this section.
- Karma Runner [defines](http://karma-runner.github.io/0.13/dev/git-commit-msg.html) the semantic commit message idea.

### Go Conventions

We follow the Go coding style standards very closely. Typically, running
`go fmt` will make your code beautiful for you.

We also typically follow the conventions recommended by `go lint` and
`govet`. Run `make test-style` to test the style conformance.

Read more:

- Effective Go [introduces formatting](https://golang.org/doc/effective_go.html#formatting).
- The Go Wiki has a great article on [formatting](https://github.com/golang/go/wiki/CodeReviewComments).

### Protobuf Conventions

Because this project is largely Go code, we format our Protobuf files as
closely to Go as possible. There are currently no real formatting rules
or guidelines for Protobuf, but as they emerge, we may opt to follow
those instead.

Standards:
- Tabs for indentation, not spaces.
- Spacing rules follow Go conventions (curly braces at line end, spaces
  around operators).

Conventions:
- Files should specify their package with `option go_package = "...";`
- Comments should translate into good Go code comments (since `protoc`
  copies comments into the destination source code file).
- RPC functions are defined in the same file as their request/response
  messages.
- Deprecated RPCs, messages, and fields are marked deprecated in the comments (`// UpdateFoo
  DEPRECATED updates a foo.`).
