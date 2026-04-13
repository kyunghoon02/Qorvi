package db

import (
	"context"
	"strings"

	"github.com/qorvi/qorvi/packages/domain"
)

type LabeledWalletGraphReader struct {
	Loader WalletGraphReader
	Labels WalletLabelReader
}

func NewLabeledWalletGraphReader(
	loader WalletGraphReader,
	labels WalletLabelReader,
) *LabeledWalletGraphReader {
	return &LabeledWalletGraphReader{
		Loader: loader,
		Labels: labels,
	}
}

func (r *LabeledWalletGraphReader) ReadWalletGraph(
	ctx context.Context,
	query WalletGraphQuery,
) (domain.WalletGraph, error) {
	if r == nil || r.Loader == nil {
		return domain.WalletGraph{}, ErrWalletGraphNotFound
	}

	graph, err := r.Loader.ReadWalletGraph(ctx, query)
	if err != nil {
		return domain.WalletGraph{}, err
	}
	if r.Labels == nil {
		applyEntityNodeFallbackLabels(&graph)
		return graph, nil
	}

	refs := make([]WalletRef, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if node.Kind != domain.WalletGraphNodeWallet || strings.TrimSpace(node.Address) == "" {
			continue
		}
		refs = append(refs, WalletRef{Chain: node.Chain, Address: node.Address})
	}

	labelsByRef, err := r.Labels.ReadWalletLabels(ctx, refs)
	if err != nil {
		return domain.WalletGraph{}, err
	}

	for index := range graph.Nodes {
		node := &graph.Nodes[index]
		switch node.Kind {
		case domain.WalletGraphNodeWallet:
			node.Labels = labelsByRef[walletRefLabelKey(WalletRef{
				Chain:   node.Chain,
				Address: node.Address,
			})]
		case domain.WalletGraphNodeEntity:
			if fallback, ok := fallbackWalletLabelFromEntity(node.ID, "", node.Label); ok {
				node.Labels = appendWalletLabelToSet(node.Labels, fallback)
				node.Labels = sortWalletLabelSet(node.Labels)
			}
		}
	}

	return graph, nil
}

func applyEntityNodeFallbackLabels(graph *domain.WalletGraph) {
	if graph == nil {
		return
	}
	for index := range graph.Nodes {
		node := &graph.Nodes[index]
		if node.Kind != domain.WalletGraphNodeEntity {
			continue
		}
		if fallback, ok := fallbackWalletLabelFromEntity(node.ID, "", node.Label); ok {
			node.Labels = appendWalletLabelToSet(node.Labels, fallback)
			node.Labels = sortWalletLabelSet(node.Labels)
		}
	}
}

var _ WalletGraphReader = (*LabeledWalletGraphReader)(nil)
