package config

import (
	"bytes"
	"sort"

	"gopkg.in/yaml.v3"
)

// Teaching comments emitted into every generated config. Kept as constants so
// they survive programmatic rewrites (e.g. `set`/`use` editing the file).
const (
	headerComment = "shutup project config — safe to commit (declarations only, no values)."
	consumesComment = "Variable NAMES this project needs. No values, no secrets — those live in\n" +
		"the env (~/.shutup/envs/). Add with `shutup set <NAME>` or `shutup use <NAME>`."
	envsComment = "Local env name -> env id. Written by `shutup init` / `env add`; you don't\n" +
		"edit these by hand. Point two projects at the same id to share an env."
	defaultEnvComment = "Env used when --env is omitted."
)

// marshal renders the config to YAML with comments and 2-space indentation.
func (c *Config) marshal() ([]byte, error) {
	doc := c.node()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Config) node() *yaml.Node {
	root := &yaml.Node{Kind: yaml.MappingNode}

	consumesKey := str("consumes")
	consumesKey.HeadComment = headerComment + "\n\n" + consumesComment
	root.Content = append(root.Content, consumesKey, c.consumesNode())

	envsKey := str("envs")
	envsKey.HeadComment = envsComment
	root.Content = append(root.Content, envsKey, c.envsNode())

	defKey := str("default_env")
	defKey.HeadComment = defaultEnvComment
	root.Content = append(root.Content, defKey, str(c.DefaultEnv))

	return root
}

func (c *Config) consumesNode() *yaml.Node {
	// Flow sequence so an empty list renders as [] and short lists stay tidy.
	n := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
	names := append([]string(nil), c.Consumes...)
	sort.Strings(names)
	for _, name := range names {
		n.Content = append(n.Content, str(name))
	}
	return n
}

func (c *Config) envsNode() *yaml.Node {
	m := &yaml.Node{Kind: yaml.MappingNode}
	names := make([]string, 0, len(c.Envs))
	for name := range c.Envs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		m.Content = append(m.Content, str(name), str(c.Envs[name]))
	}
	return m
}

func str(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}
