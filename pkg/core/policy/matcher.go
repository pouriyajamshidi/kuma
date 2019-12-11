package policy

import (
	"sort"

	mesh_proto "github.com/Kong/kuma/api/mesh/v1alpha1"
	mesh_core "github.com/Kong/kuma/pkg/core/resources/apis/mesh"
	core_xds "github.com/Kong/kuma/pkg/core/xds"
)

type ServiceIterator interface {
	Next() (core_xds.ServiceName, bool)
}

type ServiceIteratorFunc func() (core_xds.ServiceName, bool)

func (f ServiceIteratorFunc) Next() (core_xds.ServiceName, bool) {
	return f()
}

func ToOutboundServicesOf(dataplane *mesh_core.DataplaneResource) ServiceIteratorFunc {
	idx := 0
	return ServiceIteratorFunc(func() (core_xds.ServiceName, bool) {
		if len(dataplane.Spec.Networking.GetOutbound()) <= idx {
			return "", false
		}
		oface := dataplane.Spec.Networking.GetOutbound()[idx]
		idx++
		return oface.Service, true
	})
}

// SelectOutboundConnectionPolicies picks a single the most specific policy for each outbound interface of a given Dataplane.
func SelectOutboundConnectionPolicies(dataplane *mesh_core.DataplaneResource, policies []ConnectionPolicy) ConnectionPolicyMap {
	return SelectConnectionPolicies(dataplane, ToOutboundServicesOf(dataplane), policies)
}

// SelectConnectionPolicies picks a single the most specific policy applicable to a connection between a given dataplane and given destination services.
func SelectConnectionPolicies(dataplane *mesh_core.DataplaneResource, destinations ServiceIterator, policies []ConnectionPolicy) ConnectionPolicyMap {
	sort.Stable(ConnectionPolicyByName(policies)) // sort to avoid flakiness

	// First, select only those ConnectionPolicies that have a `source` selector matching a given Dataplane.
	// If a ConnectionPolicy has multiple matching `source` selectors, we need to choose the most specific one.
	// Technically, we give a rank to every matching selector. The more specific selector is, the higher rank it gets.

	type candidateBySource struct {
		policy         ConnectionPolicy
		bestSourceRank mesh_proto.TagSelectorRank
	}

	candidatesBySource := []candidateBySource{}
	for _, policy := range policies {
		candidate := candidateBySource{policy: policy}
		matches := false
		for _, source := range policy.Sources() {
			sourceSelector := mesh_proto.TagSelector(source.Match)
			if dataplane.Spec.Matches(sourceSelector) {
				sourceRank := sourceSelector.Rank()
				if !matches || sourceRank.CompareTo(candidate.bestSourceRank) > 0 {
					// TODO(yskopets): use CreationDate to resolve a conflict between 2 equal ranks
					candidate.bestSourceRank = sourceRank
				}
				matches = true
			}
		}
		if matches {
			candidatesBySource = append(candidatesBySource, candidate)
		}
	}

	// Then, for each outbound interface consider all ConnectionPolicies that match it by a `destination` selector.
	// If a ConnectionPolicy has multiple matching `destination` selectors, we need to choose the most specific one.
	//
	// It's possible that there will be multiple ConnectionPolicies that match a given outbound interface.
	// To choose between them, we need to compute an aggregate rank of the most specific selector by `source`
	// with the most specific selector by `destination`.

	type candidateByDestination struct {
		candidateBySource
		bestAggregateRank mesh_proto.TagSelectorRank
	}

	candidatesByDestination := map[core_xds.ServiceName]candidateByDestination{}
	for service, ok := destinations.Next(); ok; service, ok = destinations.Next() {
		if _, ok := candidatesByDestination[service]; ok {
			// apparently, multiple outbound interfaces of a given Dataplane refer to the same service
			continue
		}
		outboundTags := mesh_proto.SingleValueTagSet{mesh_proto.ServiceTag: service}
		for _, candidateBySource := range candidatesBySource {
			for _, destination := range candidateBySource.policy.Destinations() {
				destinationSelector := mesh_proto.TagSelector(destination.Match)
				if destinationSelector.Matches(outboundTags) {
					aggregateRank := destinationSelector.Rank().CombinedWith(candidateBySource.bestSourceRank)

					candidateByDestination, exists := candidatesByDestination[service]

					if !exists || aggregateRank.CompareTo(candidateByDestination.bestAggregateRank) > 0 {
						// TODO(yskopets): use CreationDate to resolve a conflict between 2 equal ranks
						candidateByDestination.candidateBySource = candidateBySource
						candidateByDestination.bestAggregateRank = aggregateRank

						candidatesByDestination[service] = candidateByDestination
					}
				}
			}
		}
	}

	policyMap := ConnectionPolicyMap{}
	for service, candidate := range candidatesByDestination {
		policyMap[service] = candidate.policy
	}
	return policyMap
}

type ConnectionPolicyByName []ConnectionPolicy

func (a ConnectionPolicyByName) Len() int      { return len(a) }
func (a ConnectionPolicyByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ConnectionPolicyByName) Less(i, j int) bool {
	return a[i].GetMeta().GetName() < a[j].GetMeta().GetName()
}
