
# deploy to k8s

(with cloned repo)
```
# kubectl apply -f deploy/ -n namespace
```

(without cloned repo)
```
# kubectl apply -f https://raw.githubusercontent.com/xetys/k8s-bitflow/master/deploy/rbac.yaml -n namespace
# kubectl apply -f https://raw.githubusercontent.com/xetys/k8s-bitflow/master/deploy/operator.yaml -n namespace
```

## verify

After succeeded deploy each node should run a bitflow pod and the operator, like this:

```
# kubectl get po      
NAME                                                     READY     STATUS    RESTARTS   AGE
bitflow-level-four-master-01                             1/1       Running   0          12m
bitflow-level-four-master-02                             1/1       Running   0          12m
bitflow-level-four-worker-01                             1/1       Running   0          12m
bitflow-level-four-worker-02                             1/1       Running   0          12m
bitflow-operator-7999449f6d-qbxbs                        1/1       Running   0          12m
```

## get a config from cluster

```
# kubectl get pod -l app=bitflow-operator -o="jsonpath={.items[*].metadata.name}" | awk '{print "kubectl exec " $1 " -- /k8s-bitflow gen-config"}' | sh
```

or if this tool is already built on your local machine:

```
# ./k8s-bitflow gen-config
```
