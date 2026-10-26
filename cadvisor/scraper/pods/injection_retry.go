package pods

import (
	"sync"
	"time"

	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
)

// The node helper writes the container ID for an APM agent by running `<agent> -c <containerID>` inside the
// container once it is Running. The first attempt can fail before the application has started its agent
// (the Python agent creates /whatap-agent/whatap_python at startup) or on a transient exec error. Such a
// container used to be marked processed after one failed attempt and was never retried, so container.conf
// was missing only in the pods that started slowly. A failed container is now retried with exponential
// backoff and given up after injectionMaxAttempts (about five minutes in total).
const (
	injectionRetryBaseDelay = 10 * time.Second
	injectionRetryMaxDelay  = 5 * time.Minute
	injectionMaxAttempts    = 6
	injectionRetryTick      = 5 * time.Second
)

type injectionRetry struct {
	attempts int
	nextAt   time.Time
}

var (
	injectionRetries sync.Map // containerID -> injectionRetry
	execAgentCommand = executeWhatapAgentCommand
	nowFunc          = time.Now
)

func injectionRetryDelay(attempts int) time.Duration {
	delay := injectionRetryBaseDelay
	for n := 1; n < attempts; n++ {
		delay *= 2
		if delay >= injectionRetryMaxDelay {
			return injectionRetryMaxDelay
		}
	}
	return delay
}

// recordInjectionFailure counts at most one attempt per executeInjection pass: several agent-path sources
// can fail for the same container in one pass, and only the first failure before the retry time counts.
func recordInjectionFailure(containerID string) (attempts int, retryAfter time.Duration, gaveUp bool) {
	now := nowFunc()
	var state injectionRetry
	if v, ok := injectionRetries.Load(containerID); ok {
		state = v.(injectionRetry)
		if now.Before(state.nextAt) {
			return state.attempts, state.nextAt.Sub(now), false
		}
	}
	state.attempts++
	if state.attempts >= injectionMaxAttempts {
		injectionRetries.Delete(containerID)
		processedContainers.Store(containerID, true)
		return state.attempts, 0, true
	}
	retryAfter = injectionRetryDelay(state.attempts)
	state.nextAt = now.Add(retryAfter)
	injectionRetries.Store(containerID, state)
	return state.attempts, retryAfter, false
}

func markInjectionDone(containerID string) {
	processedContainers.Store(containerID, true)
	injectionRetries.Delete(containerID)
}

func injectionRetryDue(containerID string) bool {
	v, ok := injectionRetries.Load(containerID)
	if !ok {
		return true
	}
	return !nowFunc().Before(v.(injectionRetry).nextAt)
}

func runInjectionRetrier() {
	ticker := time.NewTicker(injectionRetryTick)
	defer ticker.Stop()
	for range ticker.C {
		retryPendingInjections()
	}
}

// retryPendingInjections re-runs injection for pods that have a failed container whose retry time has come.
// The pod informer has no resync, so without this a pod that emits no further events would never retry.
func retryPendingInjections() {
	if !whatap_config.GetConfig().InjectContainerIdToApmAgentEnabled {
		return
	}
	podsMapLock.Lock()
	defer podsMapLock.Unlock()
	for _, podInfo := range podsMap {
		if !podInfo.ContainsApmAgent || !hasDueInjection(podInfo) {
			continue
		}
		podInfo := podInfo
		executeInjection(&podInfo)
	}
}

func hasDueInjection(podInfo SimplePodInfo) bool {
	for _, c := range podInfo.ContainerInfos {
		if !c.IsApm || c.ExecProcessed || c.ContainerId == "" || c.ContainerState != "Running" {
			continue
		}
		if _, pending := injectionRetries.Load(c.ContainerId); pending && injectionRetryDue(c.ContainerId) {
			return true
		}
	}
	return false
}
