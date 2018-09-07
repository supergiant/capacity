Supergiant: Capacity Service
===

<!-- Badge Links -->
<!-- [GoReportCard Widget]: https://goreportcard.com/badge/github.com/supergiant/supergiant
[GoReportCard URL]: https://goreportcard.com/report/github.com/supergiant/supergiant
[GoDoc Widget]: https://godoc.org/github.com/supergiant/supergiant?status.svg
[GoDoc URL]: https://godoc.org/github.com/supergiant/supergiant
[Travis Widget]: https://travis-ci.org/supergiant/supergiant.svg?branch=master
[Travis URL]: https://travis-ci.org/supergiant/supergiant
[Release Widget]: https://img.shields.io/github/release/supergiant/supergiant.svg
[Release URL]: https://github.com/supergiant/supergiant/releases/latest
[Coverage Status]: https://coveralls.io/github/supergiant/supergiant?branch=master
[Coverage Status Widget]: https://coveralls.io/repos/github/supergiant/supergiant/badge.svg?branch=master -->

# <img src="http://supergiant.io/img/logo_dark.svg" width="400">

<!-- Badges -->
<!-- [![Release Widget]][Release URL] [![GoDoc Widget]][GoDoc URL] [![Travis Widget]][Travis URL] [![Coverage Status Widget]][Coverage Status] [![GoReportCard Widget]][GoReportCard URL] -->

# Features

* Autoscaling of nodes to match kube resource-requests
* Customizable profiles for many kube configurations
* A clean UI and API for interaction with various operations

# Resources

* [Official Website](https://supergiant.io/)
* [Documentation](https://supergiant.readme.io/v2.0.0/)
* [Slack Channel](https://supergiant.io/slack)

# Community and Contributing

Contributions come in a vast variety, whether docs edits, pull requests--even social media engagement--and we're thankful for community support! We only ask that users keep in mind the [Community and Contribution Guidelines](https://dash.readme.io/project/supergiant/v2.0.0/docs/guidelines), as there may be rare occasions in which we need to take action. In addtion to this, code contributions do have [a few important rules](https://supergiant.readme.io/v2.0.0/docs/guidelines#section-code-contributions) (we aren't allowed to accept code without adherence to them).

## Development

### Setup

Note: _This process is subject to change._

#### 1. Install Dependencies:

* [Git](https://git-scm.com/)
* [Go](https://golang.org/) - version 1.7+ 
* [govendor](https://github.com/kardianos/govendor)
* [npm & Node.js](https://www.npmjs.com/get-npm)
* [Angular](https://cli.angular.io/) - version 5.0+

Note: _If a new package is imported and then used in Supergiant code, make sure to vendor the imports (the govendor binary is located in `$GOPATH/bin/govendor`):_

```shell
govendor add +external
```

Note: _New to any of these technologies? Here are a few basic, free resources:_
* An [introduction to Git](https://www.youtube.com/watch?v=xuB1Id2Wxak)
* A [Golang tutorial playlist](https://www.youtube.com/watch?v=G3PvTWRIhZA&list=PLQVvvaa0QuDeF3hP0wQoSxpkqgRcgxMqX)
* A [guide to using npm](https://www.youtube.com/watch?v=jHDhaSSKmB0)
* A [guide to Angular 5](https://www.youtube.com/watch?v=AaNZBrP26LQ)

#### 2. Fork the Repo

#### 3. Clone the Fork

#### 4. Setup a Config File

Copy configuration files and update them:
```
cp config/kubescaler.conf.example config/kubescaler.conf
```

### Operation

#### 1. Running the API

Run an application:
```
go run ./cmd/capacity-service/main.go  --kubescaler-config config/kubescaler.conf
```

Run with a custom UserData:
```
go run ./cmd/capacity-service/main.go  --kubescaler-config config/kubescaler.conf --user-data /userdata.txt
```

#### 2. Running the UI

Currently, Supergiant uses a UI developed with Angular 5 and higher. The Angular project directory is found in `./supergiant/cmd/ui/assets`. The old UI is accessible on port `8080` when the Supergiant Server is running.

##### 2.a. Install Node Modules

In `./supergiant/cmd/ui/assets`, run:

```shell
npm install
```

Note: _If the UI fails to initialize, package and dependency problems are often the cause. If there is a problem with the project's current `package.json` or `package-lock.json`, please open a GitHub issue._

##### 2.b. Serve the UI

Within `./supergiant/cmd/ui/assets/`, run:

```shell
ng serve
```

The UI will be accessible on port 4200 by default. The server must be running to interact properly with the UI.

## Demo

Get a kubescaler config:
```
➜ $ curl -s -XGET localhost:8081/api/v1/config | jq
{
    "sshPubKey": "",
    "clusterName": "clusterName",
    "masterPrivateAddr": "172.20.1.236",
    "kubeAPIPort": "443",
    "kubeAPIPassword": "0lLEJhRN",
    "providerName": "aws",
"provider": {
    "awsIAMRole": "kubernetes-node",
    "awsImageID": "ami-cc0900ac",
    "awsKeyID": "keyID",
    "awsKeyName": "clusterName-key",
    "awsRegion": "us-west-1",
    "awsSecretKey": "secretKey",
    "awsSecurityGroups": "sg-59aff721",
    "awsSubnetID": "subnet-7f8daa24",
    "awsTags": "KubernetesCluster=clusterName",
    "awsVolSize": "100",
    "awsVolType": "gp2"
},
    "stopped": false,
    "nodesCountMin": 0,
    "nodesCountMax": 0,
    "machineTypes": [],
    "maxMachineProvisionTime": 0
}
```

Update a kubescaler config:
```
➜ $ curl -s -XPATCH -d'{"nodesCountMin":1000}' localhost:8081/api/v1/config | jq
{
  "sshPubKey": "",
  "clusterName": "clusterName",
  "masterPrivateAddr": "172.20.1.236",
  "kubeAPIPort": "443",
  "kubeAPIPassword": "0lLEJhRN",
  "providerName": "aws",
  "provider": {
    "awsIAMRole": "kubernetes-node",
    "awsImageID": "ami-cc0900ac",
    "awsKeyID": "keyID",
    "awsKeyName": "clusterName-key",
    "awsRegion": "us-west-1",
    "awsSecretKey": "secretKey",
    "awsSecurityGroups": "sg-59aff721",
    "awsSubnetID": "subnet-7f8daa24",
    "awsTags": "KubernetesCluster=clusterName",
    "awsVolSize": "100",
    "awsVolType": "gp2"
  },
  "stopped": false,
  "nodesCountMin": 1000,
  "nodesCountMax": 0,
  "machineTypes": [],
  "maxMachineProvisionTime": 0
}
```

List all supported machine types:
```
curl -s localhost:8081/api/v1/machinetypes | jq
[
  {
    "name": "m4.large",
    "cpu": "8Gi",
    "memory": "2"
  },
  {
    "name": "m4.xlarge",
    "cpu": "16Gi",
    "memory": "4"
  }
]
```

Create a kubescaler worker:
```
➜ $ curl -s -XPOST -d'{"machineType":"m4.large"}' localhost:8081/api/v1/workers | jq
    {
      "clusterName": "clusterName",
      "machineID": "i-01e9c47fede75cb9a",
      "machineName": "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
      "machineType": "m4.large",
      "machineState": "pending",
      "createdAt": "2018-08-01T10:58:57Z",
      "nodeName": "",
      "reserved": false
    }
```

List kubescaler workers:
```
➜ $ curl -s localhost:8081/api/v1/workers | jq
    [
      {
        "clusterName": "clusterName",
        "machineID": "i-0f48e8430160865a1",
        "machineName": "clusterName-node-0cqKZ",
        "machineType": "m4.large",
        "machineState": "running",
        "createdAt": "2018-08-01T09:01:28Z",
        "nodeName": "",
        "reserved": false
      },
      {
        "clusterName": "clusterName",
        "machineID": "i-08da815ee67272756",
        "machineName": "clusterName-master",
        "machineType": "m4.large",
        "machineState": "running",
        "createdAt": "2018-08-01T09:01:11Z",
        "nodeName": "",
        "reserved": false
      },
      {
        "clusterName": "clusterName",
        "machineID": "i-01e9c47fede75cb9a",
        "machineName": "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
        "machineType": "m4.large",
        "machineState": "pending",
        "createdAt": "2018-08-01T10:58:57Z",
        "nodeName": "",
        "reserved": false
      }
    ]
```

Make a worker reserved:
```
curl -s -XPATCH -d'{"reserved":true}' localhost:8081/api/v1/workers/mocked-worker | jq
{
  "clusterName": "",
  "machineID": "mocked-worker",
  "machineName": "",
  "machineType": "",
  "machineState": "",
  "creationTimestamp": "0001-01-01T00:00:00Z",
  "nodeName": "",
  "reserved": true
}
```

Delete a kubescaler worker:
```
➜ curl -s -XDELETE localhost:8081/api/v1/workers/i-01e9c47fede75cb9a | jq
    {
      "clusterName": "fake",
      "machineID": "i-01e9c47fede75cb9a",
      "machineName": "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
      "machineType": "m4.large",
      "machineState": "terminating",
      "createdAt": "2018-08-06T14:26:27.736042905+03:00",
      "nodeName": "",
      "reserved": false
    }

```
