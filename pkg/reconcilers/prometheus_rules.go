package reconcilers

import (
	"github.com/3scale/3scale-operator/pkg/common"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

// Remove 'ThreeScaleApicastRequestTime' alert
func RemovePrometheusRulesMutator(existing, desired common.KubernetesObject) (bool, error) {
	existingPrometheusRule := existing.(*monitoringv1.PrometheusRule)
	removed := false
	updatedRules := []monitoringv1.Rule{}
	group := existingPrometheusRule.Spec.Groups[0]

	for _, rule := range group.Rules {
		if rule.Alert != "ThreescaleApicastRequestTime" {
			updatedRules = append(updatedRules, rule)
		} else {
			removed = true
		}
	}
	if removed {
		group.Rules = updatedRules
		existingPrometheusRule.Spec.Groups[0] = group

		log.Info("Alert 'ThreescaleApicastRequestTime' removed from PrometheusRules")
		return true, nil
	}
	log.Info("Alert 'ThreescaleApicastRequestTime' not found, no update required.")
	return false, nil
}
