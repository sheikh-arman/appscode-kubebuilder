/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package employee

import (
	"context"
	"fmt"
	llog "log"

	employeev1alpha1 "github.com/sheikh-arman/appscode-kubebuilder/api/employee/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// EmployeeReconciler reconciles a Employee object
type EmployeeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=employee.appscode.com,resources=employees,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=employee.appscode.com,resources=employees/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=employee.appscode.com,resources=employees/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Employee object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *EmployeeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	employee := employeev1alpha1.Employee{}
	llog.Println("\n\n\ncheck req->>>>")
	llog.Println(req.Name, req.Namespace, req.NamespacedName)
	err := r.Get(context.Background(), req.NamespacedName, &employee)
	if err != nil {
		r.deleteDeployment(req.Namespace, req.Name)
		r.deleteDeployment(req.Namespace, req.Name+"-mysql")
		r.deleteService(req.Namespace, req.Name)
		r.deleteService(req.Namespace, req.Name+"-mysql")
		r.deleteJob(req.Namespace, req.Name)
		r.deleteConfigMap(req.Namespace, req.Name)
		r.deleteConfigSecret(req.Namespace, req.Name)
		r.deletePVC(req.Namespace, "mysql-pv-claim")
		r.deleteStorageClass(req.Namespace, "standard2")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	llog.Println("test->>>>>>>>")
	llog.Println(employee.Name)
	llog.Println(employee.Namespace)
	llog.Println(employee.Spec.ApiImage)
	r.createStorageClass(req.Namespace, employee)
	r.createPVC(req.Namespace, employee)
	r.createConfigSecret(req.Namespace, employee)
	r.createConfigMap(req.Namespace, employee)
	r.createDatabaseDeployment(req.Namespace, employee)
	r.createDatabaseService(req.Namespace, employee)
	r.createJob(req.Namespace, employee)
	r.createApiDeployment(req.Namespace, employee)
	r.createApiService(req.Namespace, employee)
	//r.deleteJob(req.Namespace, employee)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmployeeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&employeev1alpha1.Employee{}).
		Complete(r)
}

func (r *EmployeeReconciler) createJob(ns string, employee employeev1alpha1.Employee) {
	//create deployment for the employee CR
	name := employee.Name
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name + "-job",
							Image: "kanisterio/mysql-sidecar:0.44.0",
							Env: []corev1.EnvVar{
								{
									Name: "DB_HOST",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
											},
											Key: "host",
										},
									},
								},
								{
									Name: "DB_NAME",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
											},
											Key: "dbname",
										},
									},
								},
								{
									Name: "MYSQL_ROOT_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: employee.Name + "-secret",
											},
											Key: "root-password",
										},
									},
								},
							},
							//Ports: []corev1.ContainerPort{
							//	{
							//		Name:          "http",
							//		Protocol:      corev1.ProtocolTCP,
							//		ContainerPort: 8080,
							//	},
							//},
							Command: []string{
								"/bin/sh",
							},
							Args: []string{
								"-c",
								"until mysql -h $DB_HOST -u root --password=$MYSQL_ROOT_PASSWORD -e 'show databases;';do echo 'waiting for db ready';sleep 3;done;echo 'Database ready';mysql -h $DB_HOST -u root --password=$MYSQL_ROOT_PASSWORD -e 'create database appscode;create table appscode.employee ( id INT AUTO_INCREMENT PRIMARY KEY, name varchar(50), salary int);insert into appscode.employee( name, salary) values (\"John Doe\", 5000);insert into appscode.employee( name, salary) values (\"James William\", 7000);select * from appscode.employee;'",
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: int32Ptr(0),
		},
	}
	llog.Printf("\n\n\n\nCreating job class for %s resource", name)
	err := r.Create(context.Background(), job)
	if err != nil {
		llog.Printf("Error on creating job %s, err: %s", job, err.Error())
		return
	}
	llog.Println("job created successfully")
}

func (r *EmployeeReconciler) createStorageClass(ns string, employee employeev1alpha1.Employee) {
	//create deployment for the employee CR
	storageClassName := "standard2"
	reclaimPolicy := corev1.PersistentVolumeReclaimDelete
	volumeBindingMode := storagev1.VolumeBindingImmediate
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: storageClassName,
		},
		Provisioner:       "kubernetes.io/host-path",
		ReclaimPolicy:     &reclaimPolicy,
		VolumeBindingMode: &volumeBindingMode,
	}
	llog.Printf("Creating storage class for %s resource", storageClassName)
	err := r.Create(context.Background(), storageClass)
	if err != nil {
		llog.Printf("Error on creating storage class %s, err: %s", storageClassName, err.Error())
		return
	}
	llog.Println("pvc created successfully")
}

func (r *EmployeeReconciler) createPVC(ns string, employee employeev1alpha1.Employee) {
	//create deployment for the employee CR
	name := "mysql-pv-claim"
	storageClassName := "standard2"
	storageQuantity, _ := resource.ParseQuantity("1Gi")
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageQuantity,
				},
			},
		},
	}
	llog.Printf("Creating pvc for %s resource", name)
	err := r.Create(context.Background(), pvc)
	if err != nil {
		llog.Printf("Error on creating pvcy %s, err: %s", name, err.Error())
		return
	}
	llog.Println("pvc created successfully")
}

func (r *EmployeeReconciler) createDatabaseDeployment(ns string, employee employeev1alpha1.Employee) {
	//create deployment for the employee CR
	name := employee.Name + "-mysql"
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(employee.Spec.DatabaseReplica)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: "Recreate",
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: employee.Spec.DatabaseImage,
							//Ports: []corev1.ContainerPort{
							//	{
							//		Name:          "http",
							//		Protocol:      corev1.ProtocolTCP,
							//		ContainerPort: 8080,
							//	},
							//},
							Env: []corev1.EnvVar{

								{
									Name: "MYSQL_ROOT_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: employee.Name + "-secret",
											},
											Key: "root-password",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          name,
									ContainerPort: 3306,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      name + "-persistent-storage",
									MountPath: "/var/lib/mysql",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: name + "-persistent-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "mysql-pv-claim",
								},
							},
						},
					},
				},
			},
		},
	}
	llog.Printf("Creating database deploy for %s resource", name)
	err := r.Create(context.Background(), deployment)
	if err != nil {
		llog.Printf("Error on creating database deploy %s, err: %s", name, err.Error())
		return
	}
	llog.Println("Database deployment created successfully")
}

func (r *EmployeeReconciler) createConfigMap(ns string, employee employeev1alpha1.Employee) {
	//create deployment for the employee CR
	name := employee.Name
	svc := employee.Name + "-mysql." + ns
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string]string{
			"dbname": "appscode",
			"host":   svc,
			"port":   "3306",
		},
	}
	llog.Printf("Creating ConfigMap for %s resource", name)
	err := r.Create(context.Background(), configMap)
	if err != nil {
		llog.Printf("Error on creating database deploy %s, err: %s", name, err.Error())
		return
	}
	llog.Println("Config map created successfully")
}

func (r *EmployeeReconciler) createConfigSecret(ns string, employee employeev1alpha1.Employee) {
	//create deployment for the employee CR
	name := employee.Name + "-secret"
	configSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"root-password": []byte("YXJtYW4="),
		},
	}
	llog.Printf("Creating ConfigSecret for %s resource", name)
	err := r.Create(context.Background(), configSecret)
	if err != nil {
		llog.Printf("Error on creating secret %s, err: %s", name, err.Error())
		return
	}
	llog.Println("Secret created successfully")
}

func (r *EmployeeReconciler) createApiDeployment(ns string, employee employeev1alpha1.Employee) {
	//create deployment for the employee CR

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      employee.Name,
			Namespace: ns,
			Labels: map[string]string{
				"app": employee.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(employee.Spec.ApiReplica)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": employee.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": employee.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  employee.Name,
							Image: employee.Spec.ApiImage,
							//Ports: []corev1.ContainerPort{
							//	{
							//		Name:          "http",
							//		Protocol:      corev1.ProtocolTCP,
							//		ContainerPort: 8080,
							//	},
							//},
							Env: []corev1.EnvVar{
								{
									Name: "DB_HOST",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: employee.Name,
											},
											Key: "host",
										},
									},
								},
								{
									Name: "DB_NAME",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: employee.Name,
											},
											Key: "dbname",
										},
									},
								},
								{
									Name: "MYSQL_ROOT_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: employee.Name + "-secret",
											},
											Key: "root-password",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	llog.Printf("Creating Deployemnet for %s resource", employee.Name)
	err := r.Create(context.Background(), deployment)
	if err != nil {
		llog.Printf("Error on creating deployment %s, err: %s", employee.Name, err.Error())
		return
	}
	llog.Println("Deployment created successfully")

}

func (r *EmployeeReconciler) createDatabaseService(ns string, employee employeev1alpha1.Employee) {
	//create deployment for the employee CR
	name := employee.Name + "-mysql"
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 3306,
				},
			},
			Selector: map[string]string{
				"app": name,
			},
			ClusterIP: corev1.ClusterIPNone,
		},
	}
	llog.Printf("Creating service for %s resource", employee.Name)
	err := r.Create(context.Background(), service)
	if err != nil {
		llog.Printf("Error on creating service %s, err: %s", employee.Name, err.Error())
		return
	}
	llog.Println("Service created successfully")

}

func (r *EmployeeReconciler) createApiService(ns string, employee employeev1alpha1.Employee) {
	//create deployment for the employee CR

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      employee.Name,
			Namespace: ns,
			Labels: map[string]string{
				"app": employee.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt32(8080),
				},
			},
			Selector: map[string]string{
				"app": employee.Name,
			},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{},
		},
	}
	llog.Printf("Creating service for %s resource", employee.Name)
	err := r.Create(context.Background(), service)
	if err != nil {
		llog.Printf("Error on creating service %s, err: %s", employee.Name, err.Error())
		return
	}
	llog.Println("Service created successfully")

}

func int32Ptr(i int32) *int32 { return &i }

func (r *EmployeeReconciler) deleteDeployment(ns, employee string) {
	dep := &appsv1.Deployment{}
	err := r.Get(context.Background(), types.NamespacedName{Name: employee, Namespace: ns}, dep)
	err = r.Delete(context.Background(), dep)
	if err != nil {
		llog.Println("error on deleting deployment, err: ", err.Error())
	}

	fmt.Println("Deleted, depName: ", employee)
}

func (r *EmployeeReconciler) deleteService(ns, employee string) {
	dep := &corev1.Service{}
	err := r.Get(context.Background(), types.NamespacedName{Name: employee, Namespace: ns}, dep)
	err = r.Delete(context.Background(), dep)
	if err != nil {
		llog.Println("error on deleting service, err: ", err.Error())
	}

	fmt.Println("Deleted, service Name: ", employee)
}

func (r *EmployeeReconciler) deleteJob(ns, employee string) {

	dep := &batchv1.Job{}
	err := r.Get(context.Background(), types.NamespacedName{Name: employee, Namespace: ns}, dep)
	err = r.Delete(context.Background(), dep)
	if err != nil {
		llog.Println("error on deleting job, err: ", err.Error())
	}

	fmt.Println("Deleted, Job Name: ", employee)
	// Delete the Pods associated with the Job
	podList := &corev1.PodList{}
	err = r.List(context.Background(), podList, client.InNamespace(ns), client.MatchingLabels{"app": employee})
	if err != nil {
		llog.Println("error on listing pods, err: ", err.Error())
		return
	}
	for _, pod := range podList.Items {
		err = r.Delete(context.Background(), &pod)
		if err != nil {
			llog.Println("error on deleting pod ", pod.Name, ", err: ", err.Error())
			continue
		}
		fmt.Println("Deleted Pod: ", pod.Name)
	}
}

func (r *EmployeeReconciler) deleteConfigMap(ns, employee string) {
	dep := &corev1.ConfigMap{}
	err := r.Get(context.Background(), types.NamespacedName{Name: employee, Namespace: ns}, dep)
	err = r.Delete(context.Background(), dep)
	if err != nil {
		llog.Println("error on deleting Configmap, err: ", err.Error())
	}

	fmt.Println("Deleted, configmap Name: ", employee)
}

func (r *EmployeeReconciler) deleteConfigSecret(ns, employee string) {
	name := employee + "-secret"
	dep := &corev1.Secret{}
	err := r.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ns}, dep)
	err = r.Delete(context.Background(), dep)
	if err != nil {
		llog.Println("error on deleting secret, err: ", err.Error())
	}

	fmt.Println("Deleted, secret Name: ", employee)
}

func (r *EmployeeReconciler) deletePVC(ns, employee string) {

	dep := &corev1.PersistentVolumeClaim{}
	err := r.Get(context.Background(), types.NamespacedName{Name: employee, Namespace: ns}, dep)
	err = r.Delete(context.Background(), dep)
	if err != nil {
		llog.Println("error on deleting PVC, err: ", err.Error())
	}

	fmt.Println("Deleted, PVC Name: ", employee)
}

func (r *EmployeeReconciler) deleteStorageClass(ns, employee string) {
	_ = ns
	dep := &storagev1.StorageClass{}
	err := r.Get(context.Background(), types.NamespacedName{Name: employee}, dep)
	err = r.Delete(context.Background(), dep)
	if err != nil {
		llog.Println("error on deleting storage class, err: ", err.Error())
	}

	fmt.Println("Deleted, storage class Name: ", employee)
}
