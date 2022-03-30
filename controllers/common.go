package controllers

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"time"
)

var canaryGVR = schema.GroupVersionResource{
	Group:    "cd.org.smart",
	Version:  "v1alpha1",
	Resource: "canaries",
}
var deployGVR = schema.GroupVersionResource{
	Group:    "apps",
	Version:  "v1",
	Resource: "deployments",
}
var serviceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "services",
}
var serviceAccountGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "serviceaccounts",
}

var serviceMonitorGVR = schema.GroupVersionResource{
	Group:    "monitoring.coreos.com",
	Version:  "v1",
	Resource: "servicemonitors",
}

const (
	Canary string = "canary"
	Stable        = "stable"
)

const (
	reSyncPeriod = 60 * time.Second
)

const (
	Istio   string = "istio"
	Traefik        = "traefik"
	Nginx          = "nginx"
)
