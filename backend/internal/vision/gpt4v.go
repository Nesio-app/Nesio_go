package vision

import "strings"

// BuildItemExtractionPrompt returns a stable prompt for image-to-item extraction.
func BuildItemExtractionPrompt(locale string) string {
	if strings.TrimSpace(locale) == "" {
		locale = "zh"
	}
	return "You are an item recognition expert. Analyze one image and return JSON only with keys: " +
		"name, category, brand, color, quantity, unit, expiry_date, is_document, document_type, document_number, " +
		"suggested_room, suggested_container, tags, description, locale. locale=" + locale
}

// NormalizeExtraction ensures required keys are present for downstream code.
func NormalizeExtraction(in map[string]any, locale string) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	if strings.TrimSpace(locale) == "" {
		locale = "zh"
	}
	if _, ok := out["name"]; !ok {
		out["name"] = "新物品"
	}
	if _, ok := out["category"]; !ok {
		out["category"] = "other"
	}
	if _, ok := out["quantity"]; !ok {
		out["quantity"] = 1
	}
	if _, ok := out["unit"]; !ok {
		out["unit"] = "piece"
	}
	if _, ok := out["tags"]; !ok {
		out["tags"] = []string{"待确认", "拍照"}
	}
	out["locale"] = locale
	return out
}
