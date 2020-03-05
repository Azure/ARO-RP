package install

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *Installer) addClusterAdminGroup(ctx context.Context) error {
	rb, err := i.kubernetescli.RbacV1().ClusterRoleBindings().Get("cluster-admins", metav1.GetOptions{})
	if err != nil {
		return err
	}
	subject := rbacv1.Subject{
		Kind: rbacv1.GroupKind,
		Name: "system:aro-service",
	}

	rb.Subjects = append(rb.Subjects, subject)
	_, err = i.kubernetescli.RbacV1().ClusterRoleBindings().Update(rb)
	if err != nil {
		return err
	}
	return nil
}
