# EDB CONTAINERS

EDB Containers provide container and kubernetes solutions to various EnterpriseDB product offerings

## Getting Started

You will need to have the following tools installed to build, run and contribute to EDB Containers:

* [docker](https://docs.docker.com/get-docker/)
* [go](https://golang.org/doc/install)
* [helm](https://helm.sh/docs/intro/install/)
* [skaffold](https://skaffold.dev/docs/install/)
* [git-flow](https://github.com/nvie/gitflow/wiki/Installation)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

### Recommended Tools

If you are going to do some operator programming:

* [operator-sdk](https://docs.openshift.com/container-platform/4.1/applications/operator_sdk/osdk-getting-started.html#osdk-installing-cli_osdk-getting-started)

Not required, but these tools will make your life easier:
* [kubectx](https://github.com/ahmetb/kubectx)
* [stern](https://github.com/wercker/stern)

### Installing -- MAC

#### Docker
Follow link instructions:
<https://docs.docker.com/docker-for-mac/install/>

#### HOMEBREW INSTALLS
```bash
brew install go
brew install helm
brew install skaffold
brew install git-flow
brew install kubectl

# go code
brew install operator-sdk

# optional
brew install stern
brew install kubectx
```

### Setup

#### Tools
Before you can build you will need to run the tool setup once:

```bash
make tools
```

#### Git Flow
Initialize gitflow in your local repository:

```bash
git flow init -d
```

#### Go Development

You may need to setup your GOPATH environment variable: 

EXAMPLE .bash_profile
```bash
export GOPATH=<YOUR_EDB_CONTAINERS_REBOOT_DIRECTORY>/go
```

## Building

```bash
make build
```
