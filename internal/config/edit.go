package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AddNetworkDomains adds domains to the network: allowlist in the YAML file at
// path, preserving comments and key order. Domains already present are skipped.
// The network: key is created if absent.
func AddNetworkDomains(path string, domains []string) error {
	return editNetwork(path, domains, nil)
}

// RemoveNetworkDomains removes domains from the network: allowlist in the YAML
// file at path, preserving comments and key order. Domains not present are
// ignored. The network: key is retained even if the list becomes empty.
func RemoveNetworkDomains(path string, domains []string) error {
	return editNetwork(path, nil, domains)
}

func editNetwork(path string, add, remove []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read config file: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot stat config file: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("invalid YAML in %s: %w", path, err)
	}

	mapping := documentMapping(&doc)
	seq := networkSequence(mapping)

	if len(remove) > 0 {
		removeSet := make(map[string]bool, len(remove))
		for _, d := range remove {
			removeSet[d] = true
		}
		kept := seq.Content[:0]
		for _, n := range seq.Content {
			if !removeSet[n.Value] {
				kept = append(kept, n)
			}
		}
		seq.Content = kept
	}

	for _, d := range add {
		if containsScalar(seq, d) {
			continue
		}
		seq.Content = append(seq.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: d,
		})
	}

	// Force block style so the list renders one item per line, not [a, b].
	seq.Style = 0

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("cannot flush config encoder: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), info.Mode().Perm()); err != nil {
		return fmt.Errorf("cannot write config file: %w", err)
	}
	return nil
}

// documentMapping returns the top-level mapping node, creating the document and
// mapping if the file was empty.
func documentMapping(doc *yaml.Node) *yaml.Node {
	if len(doc.Content) == 0 {
		mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{mapping}
		return mapping
	}
	return doc.Content[0]
}

// networkSequence returns the sequence node for the network: key, creating the
// key with an empty sequence if absent, or converting a null/scalar value into
// a sequence node.
func networkSequence(mapping *yaml.Node) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == "network" {
			val := mapping.Content[i+1]
			if val.Kind != yaml.SequenceNode {
				val.Kind = yaml.SequenceNode
				val.Tag = "!!seq"
				val.Value = ""
				val.Content = nil
			}
			return val
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "network"}
	seqNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	mapping.Content = append(mapping.Content, keyNode, seqNode)
	return seqNode
}

func containsScalar(seq *yaml.Node, value string) bool {
	for _, n := range seq.Content {
		if n.Value == value {
			return true
		}
	}
	return false
}
