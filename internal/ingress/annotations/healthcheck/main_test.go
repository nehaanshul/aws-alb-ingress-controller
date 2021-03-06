/*
Copyright 2016 The Kubernetes Authors.

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

package healthcheck

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/ingress/annotations/parser"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/ingress/controller/config"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/ingress/resolver"
	api "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func buildIngress() *extensions.Ingress {
	defaultBackend := extensions.IngressBackend{
		ServiceName: "default-backend",
		ServicePort: intstr.FromInt(80),
	}

	return &extensions.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
		Spec: extensions.IngressSpec{
			Backend: &extensions.IngressBackend{
				ServiceName: "default-backend",
				ServicePort: intstr.FromInt(80),
			},
			Rules: []extensions.IngressRule{
				{
					Host: "foo.bar.com",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								{
									Path:    "/foo",
									Backend: defaultBackend,
								},
							},
						},
					},
				},
			},
		},
	}
}

type mockBackend struct {
	resolver.Mock
}

func TestIngressHealthCheck(t *testing.T) {
	ing := buildIngress()

	data := map[string]string{}
	data[parser.GetAnnotationWithPrefix("healthcheck-interval-seconds")] = "15"
	ing.SetAnnotations(data)

	hzi, _ := NewParser(mockBackend{}).Parse(ing)
	hz, ok := hzi.(*Config)
	if !ok {
		t.Errorf("expected a Upstream type")
	}

	if *hz.IntervalSeconds != 15 {
		t.Errorf("expected 2 as healthcheck-interval-seconds but returned %v", *hz.IntervalSeconds)
	}

	if *hz.Path != "/" {
		t.Errorf("expected 0 as healthcheck-path but returned %v", hz.Path)
	}
}

func TestMerge(t *testing.T) {
	for _, tc := range []struct {
		Source         *Config
		Target         *Config
		Config         *config.Configuration
		ExpectedResult *Config
	}{
		{
			Source: &Config{
				Path:            aws.String("PathA"),
				Port:            aws.String("PortA"),
				Protocol:        aws.String("udp"),
				IntervalSeconds: aws.Int64(42),
				TimeoutSeconds:  aws.Int64(43),
			},
			Target: &Config{
				Path:            aws.String("PathB"),
				Port:            aws.String("PortB"),
				Protocol:        aws.String("tcp"),
				IntervalSeconds: aws.Int64(52),
				TimeoutSeconds:  aws.Int64(53),
			},
			Config: &config.Configuration{
				DefaultBackendProtocol: "tcp",
			},
			ExpectedResult: &Config{
				Path:            aws.String("PathA"),
				Port:            aws.String("PortA"),
				Protocol:        aws.String("udp"),
				IntervalSeconds: aws.Int64(42),
				TimeoutSeconds:  aws.Int64(43),
			},
		},
		{
			Source: &Config{
				Path:            aws.String(DefaultPath),
				Port:            aws.String(DefaultPort),
				Protocol:        aws.String("tcp"),
				IntervalSeconds: aws.Int64(DefaultIntervalSeconds),
				TimeoutSeconds:  aws.Int64(DefaultTimeoutSeconds),
			},
			Target: &Config{
				Path:            aws.String("PathB"),
				Port:            aws.String("PortB"),
				Protocol:        aws.String("udp"),
				IntervalSeconds: aws.Int64(52),
				TimeoutSeconds:  aws.Int64(53),
			},
			Config: &config.Configuration{
				DefaultBackendProtocol: "tcp",
			},
			ExpectedResult: &Config{
				Path:            aws.String("PathB"),
				Port:            aws.String("PortB"),
				Protocol:        aws.String("udp"),
				IntervalSeconds: aws.Int64(52),
				TimeoutSeconds:  aws.Int64(53),
			},
		},
	} {
		actualResult := tc.Source.Merge(tc.Target, tc.Config)
		assert.Equal(t, tc.ExpectedResult, actualResult)
	}
}
