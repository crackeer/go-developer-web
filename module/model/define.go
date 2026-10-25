package model

// ModelParse 返回一个空的业务结构体指针，供 MySQL 写入时做强类型绑定。
type ModelParse func() interface{}

var modelParseFuncMap = map[string]ModelParse{}

// RegisterModelParse 由插件在 init 中调用，为「库#表」注册强类型模型。
// 开源底座自身不注册任何业务模型。
func RegisterModelParse(database, tableName string, parseFunc ModelParse) {
	key := database + "#" + tableName
	modelParseFuncMap[key] = parseFunc
}

// GetModelParseFunc 查找已注册的模型构造函数，未注册时返回 nil（回退到通用 map 写入）。
func GetModelParseFunc(database, tableName string) interface{} {
	key := database + "#" + tableName
	if parseFunc, ok := modelParseFuncMap[key]; ok {
		return parseFunc()
	}
	return nil
}
