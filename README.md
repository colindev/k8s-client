
### 步驟一 clone repo

```sh
gcloud source repos clone --project=rd-resources k8s-client
```

### 步驟二 deploy

```sh
kubectl config use-context [ your cluster context ]
kubectl apply -k k8s-client/k8s
```

### 步驟三 view log from stdout

```sh
kubectl logs -f `kubectl get po -l app=k8s-client --output jsonpath='{.items[0].metadata.name}'`
```
