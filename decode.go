package goini

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

func mapToStruct(key string, srcData map[string]interface{}, targetObj interface{}) error {
	objV := reflect.ValueOf(targetObj)
	objT := reflect.TypeOf(targetObj)

	if objT.Kind() == reflect.Ptr {
		objV = objV.Elem()
		objT = objT.Elem()
	} else {
		return errors.New("goini: The target are not ptr")
	}

	if objT.Kind() != reflect.Struct {
		return errors.New("goini: The target are not struct")
	}

	for i := 0; i < objT.NumField(); i++ {
		if !objV.Field(i).CanSet() {
			continue
		}

		field := objT.Field(i)

		tk := field.Type.Kind()

		mapKey := field.Name
		tag, _ := parseTag(field.Tag.Get("json"), ",")
		if tag == "-" {
			continue
		}

		if tag != "" {
			mapKey = tag
		}

		mapVal, _ := srcData[mapKey]

		// 检查具体的类型是否指针
		k := field.Type.Kind()
		if k == reflect.Ptr {
			k = field.Type.Elem().Kind()
		}

		if kv := decodeValue(mapVal, field.Type); kv.IsValid() {
			if tk == reflect.Ptr {
				// 初始化指针
				ptrKv := reflect.New(kv.Type())
				ptrKv.Elem().Set(kv)

				objV.Field(i).Set(ptrKv)

			} else {
				objV.Field(i).Set(kv)
			}

			continue
		}

		switch k {
		case reflect.Slice:
			arrTag, arrSeq := parseTag(field.Tag.Get("ini"), "=")
			seq := ","
			if arrTag == "seq" && arrSeq != "" {
				seq = string(arrSeq)
			}

			setVal, err := parseSlice(mapVal, field.Type, seq)
			if err != nil {
				break
			}

			objV.Field(i).Set(setVal)
		case reflect.Array:
			// todo
			break
		case reflect.Map:
			setVal, err := parseMap(mapVal, field.Type)
			if err != nil {
				break
			}

			objV.Field(i).Set(setVal)
		default:
			if objV.IsValid() {
				value := objV.Field(i)
				val := value.Interface()
				if value.Kind() == reflect.Ptr {
					if value.IsNil() {
						objV.Field(i).Set(reflect.New(field.Type.Elem()))
					}

					val = objV.Field(i).Interface()
				} else if value.Kind() == reflect.Struct {
					val = value.Addr().Interface()
				}

				if key == "" {
					mapKey = strings.ToLower(objV.Type().Name()) + "." + mapKey
				}

				nextData := srcData[mapKey]

				if nextMap, ok := nextData.(map[string]interface{}); ok {
					mapToStruct(mapKey, nextMap, val)
				}
			}
		}
	}

	return nil
}

func parseInt(val interface{}) (int64, error) {
	if valStr, ok := val.(string); ok {
		if floatVal, err := strconv.ParseFloat(valStr, 64); err == nil {
			return int64(floatVal), nil
		} else {
			return 0, err
		}
	}

	return 0, errors.New("goini: string assert error")
}

func parseUint(val interface{}) (uint64, error) {
	if valStr, ok := val.(string); ok {
		if floatVal, err := strconv.ParseFloat(valStr, 64); err == nil {
			return uint64(floatVal), nil
		} else {
			return 0, err
		}
	}

	return 0, errors.New("goini: string assert error")
}

func parseFloat(val interface{}) (float64, error) {
	if valStr, ok := val.(string); ok {
		if floatVal, err := strconv.ParseFloat(valStr, 64); err == nil {
			return floatVal, nil
		} else {
			return 0, err
		}
	}

	return 0, errors.New("goini: string assert error")
}

func parseInterface(val interface{}) interface{} {
	return val
}

func parseBool(val interface{}) (bool, error) {
	if valStr, ok := val.(string); ok {
		// 补充一些常用的词
		switch valStr {
		case "y", "Y", "on", "ON", "On", "yes", "YES", "Yes", "enabled", "ENABLED", "Enabled":
			valStr = "true"
		case "n", "N", "off", "OFF", "Off", "no", "NO", "No", "disabled", "DISABLED", "Disabled":
			valStr = "false"
		}

		if boolVal, err := strconv.ParseBool(valStr); err == nil {
			return boolVal, nil
		}
	}

	return false, errors.New("goini: string assert error")
}

func parseSlice(val interface{}, t reflect.Type, delimiter string) (reflect.Value, error) {
	if valStr, ok := val.(string); ok {
		strArr := strings.Split(valStr, delimiter)
		iL := len(strArr)

		// 初始化切片
		arr := reflect.MakeSlice(t, iL, iL)

		if iL > 0 {
			var indexT reflect.Type
			for i := 0; i < arr.Len(); i++ {
				if indexT == nil {
					indexT = arr.Index(i).Type()
				}

				// 根据具体的类型设置对应的值
				if kv := decodeValue(strArr[i], indexT); kv.IsValid() {
					if indexT.Kind() == reflect.Ptr {
						// 初始化指针
						ptrKv := reflect.New(kv.Type())
						ptrKv.Elem().Set(kv)

						arr.Index(i).Set(ptrKv)
					} else {
						arr.Index(i).Set(kv)
					}
				}
			}
		}

		return arr, nil
	}

	return reflect.MakeSlice(t, 0, 0), errors.New("goini: string assert error")
}

func parseMap(val interface{}, t reflect.Type) (reflect.Value, error) {
	m := reflect.MakeMap(t)

	if valMap, valMapOk := val.(map[string]interface{}); valMapOk {
		for k, v := range valMap {
			if vStr, ok := v.(string); ok {
				if kv := decodeValue(vStr, t.Elem()); kv.IsValid() {
					if t.Elem().Kind() == reflect.Ptr {
						// 初始化指针
						ptrKv := reflect.New(kv.Type())
						ptrKv.Elem().Set(kv)

						m.SetMapIndex(reflect.ValueOf(k), ptrKv)

					} else {
						m.SetMapIndex(reflect.ValueOf(k), kv)
					}
				}
			}
		}
	}

	return m, nil
}

func decodeValue(v interface{}, t reflect.Type) reflect.Value {
	var kv reflect.Value

	// 检查具体的类型
	k := t.Kind()
	if k == reflect.Ptr {
		k = t.Elem().Kind()
	}

	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		setVal, err := parseInt(v)
		if err != nil {
			setVal = 0
		}

		if t.Kind() == reflect.Ptr {
			kv = reflect.ValueOf(&setVal).Elem().Convert(t.Elem())
		} else {
			kv = reflect.ValueOf(setVal).Convert(t)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		setVal, err := parseUint(v)
		if err != nil {
			setVal = 0
		}

		if t.Kind() == reflect.Ptr {
			kv = reflect.ValueOf(&setVal).Elem().Convert(t.Elem())
		} else {
			kv = reflect.ValueOf(setVal).Convert(t)
		}
	case reflect.Float32, reflect.Float64:
		setVal, err := parseFloat(v)
		if err != nil {
			setVal = 0
		}

		if t.Kind() == reflect.Ptr {
			kv = reflect.ValueOf(&setVal).Elem().Convert(t.Elem())
		} else {
			kv = reflect.ValueOf(setVal).Convert(t)
		}
	case reflect.String:
		if setVal, ok := v.(string); ok {
			if t.Kind() == reflect.Ptr {
				kv = reflect.ValueOf(&setVal).Elem().Convert(t.Elem())
			} else {
				kv = reflect.ValueOf(setVal)
			}
		}
	case reflect.Interface:
		setVal := parseInterface(v)

		if t.Kind() == reflect.Ptr {
			kv = reflect.ValueOf(&setVal).Elem().Convert(t.Elem())
		} else {
			kv = reflect.ValueOf(setVal).Convert(t)
		}
	case reflect.Bool:
		setVal, err := parseBool(v)
		if err != nil {
			setVal = false
		}

		if t.Kind() == reflect.Ptr {
			kv = reflect.ValueOf(&setVal).Elem().Convert(t.Elem())
		} else {
			kv = reflect.ValueOf(setVal).Convert(t)
		}
	default:
		//其他类型暂时不处理
		return kv
	}

	return kv
}