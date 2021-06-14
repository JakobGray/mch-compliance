package cmd

import (
	"context"
	"fmt"
	"os"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	AntiAffinitySet                    = Result{Category: "AntiAffinity", Rule: "antiAffinitySet"}
	AntiAffinityLabelSet               = Result{Category: "AntiAffinity", Rule: "antiAffinityLabelSet"}
	TolerationSet                      = Result{Category: "Tolerations", Rule: "tolerationSet"}
	CustomServiceAccount               = Result{Category: "ServiceAccount", Rule: "customServiceAccount"}
	HostNetworkSetToFalse              = Result{Category: "SecurityPolicy", Rule: "hostNetworkSetToFalse"}
	HostPIDSetToFalse                  = Result{Category: "SecurityPolicy", Rule: "hostPIDSetToFalse"}
	HostIPCSetToFalse                  = Result{Category: "SecurityPolicy", Rule: "hostIPCSetToFalse"}
	RunAsNonRootSetToFalse             = Result{Category: "SecurityPolicy", Rule: "runAsNonRootSetToFalse"}
	SecurityContextSet                 = Result{Category: "SecurityPolicy", Rule: "securityContextSet"}
	AllowPrivilegeEscalationSetToFalse = Result{Category: "SecurityPolicy", Rule: "allowPrivilegeEscalationSetToFalse"}
	PrivilegedSetToFalse               = Result{Category: "SecurityPolicy", Rule: "privilegedSetToFalse"}
	ReadOnlyRootFilesystemSetToTrue    = Result{Category: "SecurityPolicy", Rule: "readOnlyRootFilesystemSetToTrue"}
	CapabilitiesDropped                = Result{Category: "SecurityPolicy", Rule: "capabilitiesDropped"}
)

type Checklist struct {
	Deployments []DeploymentCheck `json:"deployments"`
}

type DeploymentCheck struct {
	Name   string   `json:"name"`
	Checks []Result `json:"failed_checks"`
}

type Result struct {
	Category string `json:"category"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
}

func (r Result) SetMessage(m string) Result {
	r.Message = m
	return r
}

type DeploymentCheckFunc func(deployment appsv1.Deployment) []Result

var deploymentCheckFuncs = []DeploymentCheckFunc{
	checkTolerations,
	checkAntiAffinity,
	checkServiceAccount,
	checkSecurityPolicy,
}

func (c *Config) checkCompliance() (Checklist, error) {
	resultList := Checklist{}

	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("failed to create client")
		os.Exit(1)
	}

	// Deployment checks
	deployList := &appsv1.DeploymentList{}
	err = cl.List(context.Background(), deployList, client.InNamespace(namespace))
	if err != nil {
		return resultList, err
	}

	for _, d := range deployList.Items {
		dc := c.runDeploymentChecks(d)
		resultList.Deployments = append(resultList.Deployments, dc)
	}

	return resultList, nil
}

func (c *Config) runDeploymentChecks(d appsv1.Deployment) DeploymentCheck {
	errs := []Result{}
	for _, f := range deploymentCheckFuncs {
		if e := f(d); e != nil {
			errs = append(errs, e...)
		}
	}

	skips := c.Exemptions["deployments"]
	if skips != nil {
		for _, ex := range skips {
			if ex.Name == d.Name {
				errs = filterResults(errs, ex.Checks)
			}
		}
	}

	return DeploymentCheck{
		Name:   d.Name,
		Checks: errs,
	}
}

func filterResults(results []Result, skips []string) []Result {
	unskipped := []Result{}
	for _, s := range skips {
		// '*' means filter all results
		if s == "*" {
			return []Result{}
		}
	}

	for _, r := range results {
		for _, s := range skips {
			if r.Category != s && r.Rule != s {
				unskipped = append(unskipped, r)
			}
		}
	}
	return unskipped
}

func getAA(value string) *corev1.PodAntiAffinity {
	return &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{
				PodAffinityTerm: corev1.PodAffinityTerm{
					TopologyKey: "kubernetes.io/hostname",
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "ocm-antiaffinity-selector",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{value},
							},
						},
					},
				},
				Weight: 35,
			},
			{
				PodAffinityTerm: corev1.PodAffinityTerm{
					TopologyKey: "topology.kubernetes.io/zone",
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "ocm-antiaffinity-selector",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{value},
							},
						},
					},
				},
				Weight: 70,
			},
		},
	}
}

func hasAntiAffinity(have, want *corev1.PodAntiAffinity) []Result {
	errs := []Result{}
	for _, wantTerm := range want.PreferredDuringSchedulingIgnoredDuringExecution {
		found := false
		for _, haveTerm := range have.PreferredDuringSchedulingIgnoredDuringExecution {
			if haveTerm.Weight != wantTerm.Weight {
				continue
			}
			if haveTerm.PodAffinityTerm.TopologyKey != wantTerm.PodAffinityTerm.TopologyKey {
				continue
			}
			if haveTerm.PodAffinityTerm.LabelSelector == nil {
				continue
			}
			for _, haveLSR := range haveTerm.PodAffinityTerm.LabelSelector.MatchExpressions {
				wantLSR := wantTerm.PodAffinityTerm.LabelSelector.MatchExpressions[0]
				if reflect.DeepEqual(haveLSR, wantLSR) {
					found = true
				}
			}
		}
		if found == false {
			errs = append(errs, AntiAffinitySet.SetMessage(fmt.Sprintf("missing podAntiAffinity %s", wantTerm.PodAffinityTerm.TopologyKey)))
		}
	}
	return errs
}

func checkTolerations(d appsv1.Deployment) []Result {
	errs := []Result{}
	tolerations := []corev1.Toleration{
		{
			Key:      "node-role.kubernetes.io/infra",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}
	for _, t := range tolerations {
		found := false
		for _, tt := range d.Spec.Template.Spec.Tolerations {
			if t == tt {
				found = true
			}
		}
		if found == false {
			errs = append(errs, TolerationSet.SetMessage(fmt.Sprintf("missing toleration %s", t.Key)))
		}
	}
	return errs
}

func checkServiceAccount(d appsv1.Deployment) []Result {
	if d.Spec.Template.Spec.ServiceAccountName == "" {
		return []Result{
			CustomServiceAccount.SetMessage(fmt.Sprintf("using the default service account")),
		}
	}
	return []Result{}
}

func checkAntiAffinity(d appsv1.Deployment) []Result {
	errs := []Result{}
	selector, ok := d.Spec.Template.Labels["ocm-antiaffinity-selector"]
	if !ok {
		errs = append(errs, AntiAffinityLabelSet.SetMessage("missing the `ocm-antiaffinity-selector` label"))
	}

	affinity := d.Spec.Template.Spec.Affinity
	if affinity != nil {
		antiAffinity := d.Spec.Template.Spec.Affinity.PodAntiAffinity
		if antiAffinity != nil {
			errs = append(errs, hasAntiAffinity(antiAffinity, getAA(selector))...)
		} else {
			errs = append(errs, AntiAffinitySet.SetMessage("missing pod anti-affinity field"))
		}
	} else {
		errs = append(errs, AntiAffinityLabelSet.SetMessage("missing pod affinity field"))
	}

	return errs
}

func checkSecurityPolicy(d appsv1.Deployment) []Result {
	errs := []Result{}

	if d.Spec.Template.Spec.HostNetwork != false {
		errs = append(errs, HostNetworkSetToFalse.SetMessage("HostNetwork is not false"))
	}
	if d.Spec.Template.Spec.HostPID != false {
		errs = append(errs, HostPIDSetToFalse.SetMessage("HostPID is not false"))
	}
	if d.Spec.Template.Spec.HostIPC != false {
		errs = append(errs, HostIPCSetToFalse.SetMessage("HostIPC is not false"))
	}
	sc := d.Spec.Template.Spec.SecurityContext
	if sc != nil {
		if sc.RunAsNonRoot == nil || *sc.RunAsNonRoot != true {
			errs = append(errs, RunAsNonRootSetToFalse.SetMessage("RunAsNonRoot is not true"))
		}
	}

	for _, container := range d.Spec.Template.Spec.Containers {
		ctx := container.SecurityContext
		if ctx == nil {
			errs = append(errs, SecurityContextSet.SetMessage(fmt.Sprintf("Container '%s': missing SecurityContext", container.Name)))
			continue
		}

		if ctx.AllowPrivilegeEscalation == nil || *ctx.AllowPrivilegeEscalation != false {
			errs = append(errs, AllowPrivilegeEscalationSetToFalse.SetMessage(fmt.Sprintf("Container '%s': AllowPrivilegeEscalation is not false", container.Name)))
		}
		if ctx.Privileged == nil || *ctx.Privileged != false {
			errs = append(errs, PrivilegedSetToFalse.SetMessage(fmt.Sprintf("Container '%s': Privileged is not false", container.Name)))
		}
		if ctx.ReadOnlyRootFilesystem == nil || *ctx.ReadOnlyRootFilesystem != true {
			errs = append(errs, ReadOnlyRootFilesystemSetToTrue.SetMessage(fmt.Sprintf("Container '%s': ReadOnlyRootFilesystem is not true", container.Name)))
		}
		if ctx.Capabilities == nil || len(ctx.Capabilities.Drop) == 0 || ctx.Capabilities.Drop[0] != "ALL" {
			errs = append(errs, CapabilitiesDropped.SetMessage(fmt.Sprintf("Container '%s': Capabilities not dropped", container.Name)))
		}
	}

	return errs
}
