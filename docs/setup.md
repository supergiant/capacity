# Setup

## Config files

Capacity service requires kubernetes and kubescaler configuration files.

In case of deploing Capacity to the cluster, kubeconfig file is optional, but it needs a properly configured serviceaccount for RBAC. Also, kubescaler config sould be stored as ConfigMap with `kubescaler.conf` key. ([reason](https://github.com/kubernetes/kubernetes/issues/60814))

### examples

`kubeconfig` file (if rbac is enabled in cluster, capacity-user should has proper [permissions](#rbac-permissions)):
```
apiVersion: v1
clusters:
- cluster:
    certificate-authority: /tmp/ca.crt
    server: https://171.14.15.48
  name: supergiant
contexts:
- context:
    cluster: supergiant
    user: capacity-user
  name: supergiant
current-context: supergiant
kind: Config
preferences: {}
users:
- name: capacity-user
  user:
    client-certificate: /tmp/client.crt
    client-key: /tmp/client.key

```

`kubescaler.conf` file:
```
{
  "clusterName": "supergiant-cluster",
  "providerName": "aws",
  "provider": {
    "awsIAMRole": "kubernetes-node",
    "awsImageID": "ami-cc0900ac",
    "awsKeyID": "KEY_ID",
    "awsKeyName": "supergiant-cluster-key",
    "awsRegion": "us-west-1",
    "awsSecretKey": "SECRET_KEY",
    "awsSecurityGroups": "sg-05798f85745b810e6",
    "awsSubnetID": "subnet-0dd9802be57d03031",
    "awsVolSize": "100",
    "awsVolType": "gp2"
  },
  "paused": true,
  "pauseLock": true,
  "workersCountMin": 1,
  "workersCountMax": 3,
  "machineTypes": [
    "t2.micro"
  ],
  "userdata": "a base64 encoded provisioning script or cloud-init configuration"
}
```

## Out of cluster

Using the above files, command to run:
```
./capacity --kubeconfig kubeconfig --kubescaler-config kubescaler.conf
INFO[0000] setup kubescaler...                          
INFO[0000] kubescaler: get config from: "kubescaler.conf" file 
INFO[0036] starting kubescaler...                       
WARN[0036] Pause Lock engaged. Automatic Capacity will not occur. 
INFO[0036] capacityservice: listen on ":8081" 
```

## In cluster

Create the `capacity-config` configmap with the `kubescaler.conf` file:
```
kubectl create configmap capacity-config --from-file=kubescaler.conf=kubescaler.conf
```

Deploy Capacity to the cluster (if rbac is enabled in cluster, setup the [permissions](#rbac-permissions) before):
```
cat <<EOF | kubectl create -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: capacity-service
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: capacity-service
  labels:
    app: capacity-service
spec:
  replicas: 1
  selector:
    matchLabels:
      app: capacity-service
  template:
    metadata:
      labels:
        app: capacity-service
    spec:
      serviceAccountName: capacity-service
      containers:
      - name: capacity
        image: supergiant/capacity
        env:
          - name: CAPACITY_CONFIGMAP_NAME
            value: capacity-config
          - name: CAPACITY_CONFIGMAP_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
        ports:
        - containerPort: 8081
        resources:
          limits:
            memory: 500Mi
          requests:
            memory: 450Mi
---
apiVersion: v1
kind: Service
metadata:
  name: capacity-service
  labels:
    app: capacity-service
spec:
  selector:
    app: capacity-service
  ports:
  - port: 8081
    targetPort: 8081
EOF
```

## RBAC permissions

```
cat <<EOF | kubectl create -f -
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: capacity-configmap-updater
  namespace: kube-system
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  resourceNames: ["capacity"]
  verbs: ["get", "patch"]
---
# for updating kubescaler config on configmap
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: capacity-configmap
  namespace: kube-system
subjects:
- kind: ServiceAccount
  name: capacity-service
  namespace: default
roleRef:
  kind: Role
  name: capacity-configmap-updater
  apiGroup: rbac.authorization.k8s.io
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: capacity-configmap-updater
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  resourceNames: ["capacity-config"]
  verbs: ["get", "patch"]
---
# for updating kubescaler config on configmap
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: capacity-configmap
  namespace: default
subjects:
- kind: ServiceAccount
  name: capacity-service
  namespace: default
roleRef:
  kind: Role
  name: capacity-configmap-updater
  apiGroup: rbac.authorization.k8s.io
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: capacity-nodepod-permissions
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["*"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
---
# capacity has to have access for pods/nodes
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: capacity-permissions
  namespace: default
subjects:
- kind: ServiceAccount
  name: capacity-service
  namespace: default
roleRef:
  kind: ClusterRole
  name: capacity-nodepod-permissions
  apiGroup: rbac.authorization.k8s.io
EOF
```
