kubectl delete  configmaps appscode-cm -n appscode
kubectl delete pvc mysql-pv-claim -n appscode
kubectl delete service/appscode-mysql -n appscode
kubectl delete service/appscode -n appscode
kubectl delete storageclass standard2