package openapi

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// OperationLines finds the line of each operation in the text of a spec (YAML or JSON), so a finding
// can point at the file. The key is OperationKey(method, path). A path whose item has no operation
// of that method is not listed. It returns an empty map when the text cannot be read as a document.
func OperationLines(data []byte) map[string]int {
	lines := map[string]int{}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		return lines
	}
	root := doc.Content[0]
	paths := mappingValue(root, "paths")
	if paths == nil {
		return lines
	}
	for i := 0; i+1 < len(paths.Content); i += 2 {
		pathKey, item := paths.Content[i], paths.Content[i+1]
		if item.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(item.Content); j += 2 {
			method := strings.ToLower(item.Content[j].Value)
			switch method {
			case "get", "put", "post", "delete", "options", "head", "patch", "trace":
				lines[OperationKey(method, pathKey.Value)] = item.Content[j].Line
			}
		}
	}
	return lines
}

func mappingValue(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}
