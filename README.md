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
➜  capacity git:(capacity_app) ✗ curl -s -XGET localhost:8081/kubescaler/config | jq
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
➜  capacity git:(capacity_app) ✗ curl -s -XPATCH -d'{"nodesCountMin":1000}' localhost:8081/kubescaler/config | jq
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

List kubescaler workers:
```
➜  capacity git:(capacity_app) ✗ curl -s -XGET localhost:8081/kubescaler/workers | jq
[
  {
    "ClusterName": "clusterName",
    "MachineID": "i-0e2aac09d0774e74e",
    "MachineName": "clusterName-node-PtyG4",
    "MachineType": "m4.large",
    "CreatedAt": "2018-07-31T08:21:32Z",
    "NodeName": "",
    "Reserved": false
  },
  {
    "ClusterName": "clusterName",
    "MachineID": "i-0d31d6ae61474c7ed",
    "MachineName": "clusterName-master",
    "MachineType": "m4.large",
    "CreatedAt": "2018-07-31T08:21:13Z",
    "NodeName": "",
    "Reserved": false
  }
]
```

Create a kubescaler worker:
```
➜  capacity git:(capacity_app) ✗ curl -s -XPOST -d'{"MachineType":"t2.micro"}' localhost:8081/kubescaler/workers
```

Delete a kubescaler worker:
```
➜  capacity git:(capacity_app) ✗ curl -s -XDELETE -d'{"MachineID":"i-0e2aac09d0774e74e"}' localhost:8081/kubescaler/workers

```
