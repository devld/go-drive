package utils

import "reflect"

type VisitNode func(reflect.Value, *reflect.StructField)

func VisitValueTree(v any, fn VisitNode) any {
	val := reflect.Indirect(reflect.ValueOf(v))
	r := visitValueTree(val, nil, fn)
	if r.IsValid() {
		return r.Interface()
	}
	return nil
}

func visitValueTree(v reflect.Value, sf *reflect.StructField, fn VisitNode) reflect.Value {
	if !v.IsValid() || ((v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil()) {
		return v
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return v
		}
		el := v.Elem()
		p := visitValueTree(el, nil, fn)
		pr := reflect.New(p.Type())
		pr.Elem().Set(p)
		return pr
	case reflect.Map:
		r := reflect.MakeMapWithSize(v.Type(), v.Len())
		mapR := v.MapRange()
		for mapR.Next() {
			r.SetMapIndex(visitValueTree(mapR.Key(), nil, fn), visitValueTree(mapR.Value(), nil, fn))
		}
		return r
	case reflect.Slice:
		n := v.Len()
		r := reflect.MakeSlice(v.Type(), n, n)
		for i := 0; i < n; i++ {
			r.Index(i).Set(visitValueTree(v.Index(i), nil, fn))
		}
		return r
	case reflect.Struct:
		r := reflect.New(v.Type())
		n := v.NumField()
		failedFields := false
		for i := 0; i < n; i++ {
			f := v.Field(i)
			rf := r.Elem().Field(i)
			if !rf.CanSet() {
				failedFields = true
				// if there is any unexported field in this Struct, then just skip...
				break
			}
			sf := v.Type().Field(i)
			rf.Set(visitValueTree(f, &sf, fn))
		}
		if failedFields {
			return v
		}
		re := r.Elem()
		return re
	}
	vv := reflect.New(v.Type())
	vv.Elem().Set(v)
	fn(vv.Elem(), sf)
	return vv.Elem()
}
