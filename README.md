# RECERT

RECERT is a Kubernetes Operator that creates and renews SSL certificates issued by [Let's Encrypt](http://letsencrypt.org) "A nonprofit Certificate Authority provider."


## WHY RECERT
Traditional Certificate Authorities issue certificates by confirming that a business entity is who he says he is and that he owns and controls a particular domain.  The Certificate is created and the system administer happily installs the cert on his webserver.

With Let's Encrypt there are no humans involved.  Ownership of a domain is proven via a challenge issued by Let's Encrypt to the the webserver via a software product called certbot.  In order for Let's Encrypt to work the webserver must have a special configuration to redirect traffic to certbot AND certbot must be able to update the certificates on the webserver.

The certbot setup is very simple for traditional servers and VMs, but runs into trouble on Kubernetes due to the effermeral nature of containers.

## WHAT RECERT DOES
Recert creates a simple SSL proxy that forwards HTTPS traffic to another webserver within Kubernetes. When a Let's Encrypt challenge is issued Recert createsa Kubernetes job which will intercept the challenge and write the resulting certificate to a Kubernetes secret.  The Recert operator will then restart the SSL proxy with the certificate secrets mounted.


## INSTALLING

TODO - Write instructions on how to install the operator

## DEVELOPMENT -- GETTING STARTED 

You will need to have the following tools installed to build, run and contribute to RECERT Containers:

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
export GOPATH=<YOUR_RECERT_DIRECTORY>/go
```

## Building

```bash
make build
```
