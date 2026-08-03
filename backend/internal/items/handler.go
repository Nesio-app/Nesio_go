package items

import "strings"

func IsGenericCameraFilename(name string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(name))
	if trimmed == "" {
		return true
	}
	if trimmed == "image" || trimmed == "photo" || trimmed == "img" || trimmed == "scan" || trimmed == "camera" || trimmed == "screenshot" {
		return true
	}
	for _, prefix := range []string{"img_", "dsc_", "pxl_", "mvimg_", "wx_camera_", "camera_"} {
		if strings.HasPrefix(trimmed, prefix) {
			allDigits := true
			for _, ch := range trimmed[len(prefix):] {
				if ch < '0' || ch > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				return true
			}
		}
	}
	return false
}
