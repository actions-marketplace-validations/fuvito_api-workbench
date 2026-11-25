package scenario

import "gopkg.in/yaml.v3"

// StepLines returns the line of each step in the text of a scenario file (YAML or JSON), in step
// order, so a failure can point at the step. It returns nil when the text cannot be read.
func StepLines(data []byte) []int {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "steps" || root.Content[i+1].Kind != yaml.SequenceNode {
			continue
		}
		lines := make([]int, 0, len(root.Content[i+1].Content))
		for _, step := range root.Content[i+1].Content {
			lines = append(lines, step.Line)
		}
		return lines
	}
	return nil
}
