package utils

import "reflect"

type VisitNode func(reflect.Value, *reflect.StructField)

func VisitValueTree(v any, fn VisitNode) any {
	val := reflect.ValueOf(v)
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
	case reflect.Interface:
		p := visitValueTree(v.Elem(), sf, fn)
		r := reflect.New(v.Type()).Elem()
		r.Set(p)
		return r
	case reflect.Ptr:
		p := visitValueTree(v.Elem(), sf, fn)
		r := reflect.New(v.Type().Elem())
		r.Elem().Set(p)
		return r
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
	case reflect.Array:
		r := reflect.New(v.Type()).Elem()
		for i := 0; i < v.Len(); i++ {
			r.Index(i).Set(visitValueTree(v.Index(i), nil, fn))
		}
		return r
	case reflect.Struct:
		// Preserve the existing behavior for structs with unexported fields:
		// they are treated as opaque values.
		for i := 0; i < v.NumField(); i++ {
			if !reflect.New(v.Type()).Elem().Field(i).CanSet() {
				return v
			}
		}
		r := reflect.New(v.Type())
		n := v.NumField()
		for i := 0; i < n; i++ {
			f := v.Field(i)
			rf := r.Elem().Field(i)
			sf := v.Type().Field(i)
			rf.Set(visitValueTree(f, &sf, fn))
		}
		return r.Elem()
	}
	vv := reflect.New(v.Type())
	vv.Elem().Set(v)
	fn(vv.Elem(), sf)
	return vv.Elem()
}
