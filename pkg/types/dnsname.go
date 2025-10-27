package types

import ackec2apis "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"

type DnsNameConfiguration struct {
	Name  string
	Type  string
	Value string
	State string
}

func ToDnsNameConfiguration(in *ackec2apis.PrivateDNSNameConfiguration) DnsNameConfiguration {
	if in == nil {
		return DnsNameConfiguration{}
	}
	return DnsNameConfiguration{
		Name:  toString(in.Name),
		Type:  toString(in.Type),
		Value: toString(in.Value),
		State: toString(in.State),
	}
}

func toString(in *string) string {
	if in == nil {
		return ""
	}
	return *in
}
