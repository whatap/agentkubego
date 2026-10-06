package pods

import corev1 "k8s.io/api/core/v1"

const javaAgentPathEnv = "WHATAP_JAVA_AGENT_PATH"

func getWhatapJavaAgentPath(envs []corev1.EnvVar) string {
	for i := len(envs) - 1; i >= 0; i-- {
		env := envs[i]
		if env.Name == javaAgentPathEnv {
			return env.Value
		}
	}
	return ""
}

func needsJavaAgentRuntimeEnv(container corev1.Container) bool {
	for i := len(container.Env) - 1; i >= 0; i-- {
		env := container.Env[i]
		if env.Name == javaAgentPathEnv {
			return env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil
		}
	}
	return false
}
