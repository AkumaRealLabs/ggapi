package ratio_setting

import "strings"

const CompactModelSuffix = "-openai-compact"
const CompactWildcardModelKey = "*" + CompactModelSuffix

func WithCompactModelSuffix(modelName string) string {
	if strings.HasSuffix(modelName, CompactModelSuffix) {
		return modelName
	}
	return modelName + CompactModelSuffix
}

// WithoutCompactModelSuffix strips the synthetic compact billing suffix so
// Advanced Custom route model rules can match the client's original model name.
func WithoutCompactModelSuffix(modelName string) string {
	return strings.TrimSuffix(modelName, CompactModelSuffix)
}
