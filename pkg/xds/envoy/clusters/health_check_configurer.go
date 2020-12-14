package clusters

import (
	"encoding/hex"

	envoy_api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_type "github.com/envoyproxy/go-control-plane/envoy/type"
	"github.com/golang/protobuf/ptypes/wrappers"

	mesh_proto "github.com/kumahq/kuma/api/mesh/v1alpha1"
	mesh_core "github.com/kumahq/kuma/pkg/core/resources/apis/mesh"
)

func HealthCheck(healthCheck *mesh_core.HealthCheckResource) ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(config *ClusterBuilderConfig) {
		config.Add(&healthCheckConfigurer{
			healthCheck: healthCheck,
		})
	})
}

type healthCheckConfigurer struct {
	healthCheck *mesh_core.HealthCheckResource
}

func mapUInt32ToInt64Range(value uint32) *envoy_type.Int64Range {
	return &envoy_type.Int64Range{
		Start: int64(value),
		End:   int64(value) + 1,
	}
}

func mapHttpHeaders(
	headers []*mesh_proto.HealthCheck_Conf_Http_HeaderValueOption,
) []*envoy_core.HeaderValueOption {
	var envoyHeaders []*envoy_core.HeaderValueOption
	for _, header := range headers {
		envoyHeaders = append(envoyHeaders, &envoy_core.HeaderValueOption{
			Header: &envoy_core.HeaderValue{
				Key:   header.Header.Key,
				Value: header.Header.Value,
			},
			Append: header.Append,
		})
	}
	return envoyHeaders
}

func tcpHealthCheck(
	tcpConf *mesh_proto.HealthCheck_Conf_Tcp,
) *envoy_core.HealthCheck_TcpHealthCheck_ {
	tcpHealthCheck := envoy_core.HealthCheck_TcpHealthCheck{}

	if tcpConf.Send != nil {
		tcpHealthCheck.Send = &envoy_core.HealthCheck_Payload{
			Payload: &envoy_core.HealthCheck_Payload_Text{
				Text: hex.EncodeToString(tcpConf.Send.Value),
			},
		}
	}

	if tcpConf.Receive != nil {
		var receive []*envoy_core.HealthCheck_Payload
		for _, r := range tcpConf.Receive {
			receive = append(receive, &envoy_core.HealthCheck_Payload{
				Payload: &envoy_core.HealthCheck_Payload_Text{
					Text: hex.EncodeToString(r.Value),
				},
			})
		}
		tcpHealthCheck.Receive = receive
	}

	return &envoy_core.HealthCheck_TcpHealthCheck_{
		TcpHealthCheck: &tcpHealthCheck,
	}
}

func httpHealthCheck(
	httpConf *mesh_proto.HealthCheck_Conf_Http,
) *envoy_core.HealthCheck_HttpHealthCheck_ {
	var expectedStatuses []*envoy_type.Int64Range
	for _, status := range httpConf.ExpectedStatuses {
		expectedStatuses = append(
			expectedStatuses,
			mapUInt32ToInt64Range(status.Value),
		)
	}

	httpHealthCheck := envoy_core.HealthCheck_HttpHealthCheck{
		Path:                httpConf.Path,
		RequestHeadersToAdd: mapHttpHeaders(httpConf.RequestHeadersToAdd),
		ExpectedStatuses:    expectedStatuses,
		CodecClientType:     envoy_type.CodecClientType_HTTP2,
	}

	return &envoy_core.HealthCheck_HttpHealthCheck_{
		HttpHealthCheck: &httpHealthCheck,
	}
}

func (e *healthCheckConfigurer) Configure(cluster *envoy_api.Cluster) error {
	if e.healthCheck == nil || e.healthCheck.Spec.Conf == nil {
		return nil
	}
	activeChecks := e.healthCheck.Spec.Conf
	healthCheck := envoy_core.HealthCheck{
		HealthChecker: &envoy_core.HealthCheck_TcpHealthCheck_{
			TcpHealthCheck: &envoy_core.HealthCheck_TcpHealthCheck{},
		},
		Interval:           activeChecks.Interval,
		Timeout:            activeChecks.Timeout,
		UnhealthyThreshold: &wrappers.UInt32Value{Value: activeChecks.UnhealthyThreshold},
		HealthyThreshold:   &wrappers.UInt32Value{Value: activeChecks.HealthyThreshold},
	}

	if tcp := activeChecks.GetTcp(); tcp != nil {
		healthCheck.HealthChecker = tcpHealthCheck(tcp)
	}

	if http := activeChecks.GetHttp(); http != nil {
		healthCheck.HealthChecker = httpHealthCheck(http)
	}

	cluster.HealthChecks = append(cluster.HealthChecks, &healthCheck)
	return nil
}
