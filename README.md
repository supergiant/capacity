# capacity

## Setup

Copy configuration files and update them:
```
cp config/kube.conf.example config/kube.conf
cp config/kubescaler.conf.example config/kubescaler.conf
```

Run an application:
```

go run ./cmd/capacity-service/main.go  --kubescaler-config config/kubescaler.conf --kube-config config/kube.conf
```

## Demo

Get a kubescaler config:
```
➜ $ curl -s -XGET localhost:8081/kubescaler/config | jq
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
➜ $ curl -s -XPATCH -d'{"nodesCountMin":1000}' localhost:8081/kubescaler/config | jq
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

Create a kubescaler worker:
```
➜ $ curl -s -XPOST -d'{"machineType":"m4.large"}' localhost:8081/kubescaler/workers | jq
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
➜ $ curl -s localhost:8081/kubescaler/workers | jq
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

Delete a kubescaler worker:
```
➜ curl -s -XDELETE -d'{"MachineID":"i-01e9c47fede75cb9a"}' localhost:8081/kubescaler/workers | jq
  {
    "clusterName": "clusterName",
    "machineID": "i-01e9c47fede75cb9a",
    "machineName": "",
    "machineType": "",
    "machineState": "shutting-down",
    "createdAt": "0001-01-01T00:00:00Z",
    "nodeName": "",
    "reserved": false
  }

```
