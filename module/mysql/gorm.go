package mysql

import (
	"fmt"
	"strings"
)

// BuildQueryRaws
//
//	@param query
//	@return string
//	@return []interface{}
func BuildQueryRaws(query map[string]interface{}) (string, []interface{}) {
	queryConditions := []string{}
	params := []interface{}{}
	for key, val := range query {

		if strings.HasPrefix(key, "like@") || strings.HasPrefix(key, "plike@") {
		} else {
			params = append(params, val)
		}

		if !strings.Contains(key, "@") {
			queryConditions = append(queryConditions, fmt.Sprintf("%s in (?)", key))
			continue

		}
		parts := strings.Split(key, "@")
		if len(parts) < 2 {
			queryConditions = append(queryConditions, fmt.Sprintf("%s in (?)", key))
			continue
		}

		switch parts[0] {
		case "not":
			queryConditions = append(queryConditions, fmt.Sprintf("%s not in (?)", parts[1]))
		case "no":
			queryConditions = append(queryConditions, fmt.Sprintf("%s != ?", parts[1]))
		case "gt":
			queryConditions = append(queryConditions, fmt.Sprintf("%s > ?", parts[1]))
		case "gte":
			queryConditions = append(queryConditions, fmt.Sprintf("%s >= ?", parts[1]))
		case "lt":
			queryConditions = append(queryConditions, fmt.Sprintf("%s < ?", parts[1]))
		case "lte":
			queryConditions = append(queryConditions, fmt.Sprintf("%s <= ?", parts[1]))
		case "like":
			queryConditions = append(queryConditions, fmt.Sprintf("%s like '%%%v%%'", parts[1], val))
		case "plike":
			queryConditions = append(queryConditions, fmt.Sprintf("%s like '%v%%'", parts[1], val))
		default:
			queryConditions = append(queryConditions, fmt.Sprintf("%s in (?)", key))
		}
	}
	return strings.Join(queryConditions, " and "), params
}
