<!-- Badge Links -->
[Release Widget]: https://img.shields.io/github/release/supergiant/capacity.svg
[Release URL]: https://github.com/supergiant/capacity/releases/latest

[GoDoc Widget]: https://godoc.org/github.com/supergiant/capacity?status.svg
[GoDoc URL]: https://godoc.org/github.com/supergiant/capacity

[Travis Widget]: https://travis-ci.org/supergiant/capacity.svg?branch=master
[Travis URL]: https://travis-ci.org/supergiant/capacity

[Coverage Status]: https://coveralls.io/github/supergiant/capacity?branch=master
[Coverage Status Widget]: https://coveralls.io/repos/github/supergiant/capacity/badge.svg?branch=master

[GoReportCard Widget]: https://goreportcard.com/badge/github.com/supergiant/capacity
[GoReportCard URL]: https://goreportcard.com/report/github.com/supergiant/capacity

<!-- Badges -->
[![Release Widget]][Release URL] [![GoDoc Widget]][GoDoc URL] [![Travis Widget]][Travis URL] [![Coverage Status Widget]][Coverage Status] [![GoReportCard Widget]][GoReportCard URL]

# capacity

## Setup

Copy configuration files and update them:
```
cp config/kubescaler.conf.example config/kubescaler.conf
```

Run an application:
```

go run ./cmd/capacity-service/main.go  --kubescaler-config config/kubescaler.conf
```

Run with a custom UserData:
```

go run ./cmd/capacity-service/main.go  --kubescaler-config config/kubescaler.conf --user-data /userdata.txt
```

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

Get a kubescaler worker:
```
➜ $ curl -s -XGET localhost:8081/api/v1/workers/i-01e9c47fede75cb9a | jq
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
