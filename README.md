# NodeSimulator

NodeSimulator is a simulator, which can simulate the node resources and state 
in kubernetes and simulate the state of pod.

## Preparations
- kubernetes v1.18+
## Deploy NodeSimulator
```shell script
kubectl apply -f https://raw.githubusercontent.com/NJUPT-ISL/NodeSimulator/master/deploy/deploy.yaml
```


## Simulate Node

- Create 2 Nodes with 1 core, 4G memory & 2 GPUs in Cluster.
```yaml
apiVersion: sim.k8s.io/v1
kind: NodeSimulator
metadata:
  name: fake-node
spec:
  number: 2
  capacity:
    cpu: "1"
    ephemeral-storage: 51539404Ki
    memory: 4Gi
    pods: "61"
    gpu:  "2"
  podCIDRs:
    - 172.16.0.64/26
  addresses:
    - address: 172.17.0.5
      type: InternalIP
```

- Create a fake pod managed by nodesimulator
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-1
  labels:
    sim.k8s.io/managed: "true"
spec:
  containers:
    - image: nginx
      name: nginx
```

## Contact us

#### QQ Group: 1048469440
![img](https://github.com/NJUPT-ISL/Breakfast/blob/master/img/qrcode_1581334380545.jpg)