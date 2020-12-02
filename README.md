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

- Create 100 Nodes with 20 core, 512G memory & 4 GPUs in Cluster.
```yaml
apiVersion: sim.k8s.io/v1
kind: NodeSimulator
metadata:
  name: titan-node
spec:
  cpu: "20k"
  memory: "512Gi"
  prefix: "test"
  podNumber: "100"
  podCidr: "172.12.1.0/8"
  number: 100
  gpu:
    number: 4
    core: "3200"
    memory: "32000"
    bandwidth: "5000"
```

## Contact us

#### QQ Group: 1048469440
![img](https://github.com/NJUPT-ISL/Breakfast/blob/master/img/qrcode_1581334380545.jpg)