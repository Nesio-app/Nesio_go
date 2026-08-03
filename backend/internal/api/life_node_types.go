package api

const (
	lifeNodeTypeTask = "task"
	lifeNodeTypeThing = "thing"
	lifeNodeTypeMemory = "memory"
	lifeNodeTypeMind = "mind"
	lifeNodeTypePerson = "person"
	lifeNodeTypeCollection = "collection"
	lifeNodeTypeEvent = "event"
)

func lifeNodeTypeMatchesMemory() string {
	return "(type = 'memory' OR type = 'mind')"
}

func lifeNodeTypeForIntent(intent string) string {
	switch intent {
	case "task":
		return lifeNodeTypeTask
	case "item":
		return lifeNodeTypeThing
	case "person":
		return lifeNodeTypePerson
	case "event":
		return lifeNodeTypeEvent
	case "collection":
		return lifeNodeTypeCollection
	default:
		return lifeNodeTypeMind
	}
}

func lifeNodeIntentLabel(intent string) string {
	switch intent {
	case "task":
		return "任务"
	case "reminder":
		return "提醒"
	case "query":
		return "检索"
	case "item":
		return "物品"
	case "person":
		return "人物"
	case "event":
		return "事件"
	case "collection":
		return "集合"
	default:
		return "记忆"
	}
}
