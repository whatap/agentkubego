package pods

import (
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestJavaPathLastLiteralAvoidsRuntime(t *testing.T) {
	c := corev1.Container{Env: []corev1.EnvVar{
		{Name: javaAgentPathEnv, ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{Key: "path"}}},
		{Name: javaAgentPathEnv, Value: "/literal/whatap.agent.jar"},
	}}
	if got := getWhatapJavaAgentPath(c.Env); got != "/literal/whatap.agent.jar" {
		t.Fatalf("path=%q", got)
	}
	if needsJavaAgentRuntimeEnv(c) {
		t.Fatal("last literal must avoid runtime query")
	}
}

func TestJavaPathLastOptionalReferenceRequiresRuntime(t *testing.T) {
	optional := true
	c := corev1.Container{Env: []corev1.EnvVar{
		{Name: javaAgentPathEnv, Value: "/earlier/whatap.agent.jar"},
		{Name: javaAgentPathEnv, ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{Key: "path", Optional: &optional}}},
	}}
	if got := getWhatapJavaAgentPath(c.Env); got != "" {
		t.Fatalf("must not assume earlier literal survived optional source: %q", got)
	}
	if !needsJavaAgentRuntimeEnv(c) {
		t.Fatal("optional reference requires resolved runtime value")
	}
}
