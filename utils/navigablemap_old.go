/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package utils

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"
)

// dataStorage is the DataProvider that can be updated
type dataStorage interface {
	DataProvider

	Set(fldPath []string, val interface{}) error
	Remove(fldPath []string) error
	GetKeys(nesteed bool) []string
}

// NavigableMap is the basic dataStorage
type NavigableMap map[string]interface{}

// String returns the map as json string
func (ms NavigableMap) String() string { return ToJSON(ms) }

// FieldAsInterface returns the value from the path
func (ms NavigableMap) FieldAsInterface(fldPath []string) (val interface{}, err error) {
	if len(fldPath) == 0 {
		err = errors.New("empty field path")
		return

	}
	opath, indx := GetPathIndex(fldPath[0])
	var has bool
	if val, has = ms[opath]; !has {
		err = ErrNotFound
		return
	}
	if len(fldPath) == 1 {
		if indx == nil {
			return

		}
		switch rv := val.(type) {
		case []string:
			if len(rv) <= *indx {
				return nil, ErrNotFound
			}
			val = rv[*indx]
			return
		case []interface{}:
			if len(rv) <= *indx {
				return nil, ErrNotFound
			}
			val = rv[*indx]
			return
		default:
		}
		// only if all above fails use reflect:
		vr := reflect.ValueOf(val)
		if vr.Kind() == reflect.Ptr {
			vr = vr.Elem()
		}
		if vr.Kind() != reflect.Slice && vr.Kind() != reflect.Array {
			return nil, ErrNotFound

		}
		if *indx >= vr.Len() {
			return nil, ErrNotFound
		}
		return vr.Index(*indx).Interface(), nil

	}
	if indx == nil {
		switch dp := ms[fldPath[0]].(type) {
		case DataProvider:
			return dp.FieldAsInterface(fldPath[1:])
		case map[string]interface{}:
			return NavigableMap(dp).FieldAsInterface(fldPath[1:])
		default:
			err = fmt.Errorf("Wrong path")

			return
		}
	}
	switch dp := ms[opath].(type) {
	case []DataProvider:
		if len(dp) <= *indx {
			return nil, ErrNotFound
		}
		return dp[*indx].FieldAsInterface(fldPath[1:])
	case []NavigableMap:
		if len(dp) <= *indx {
			return nil, ErrNotFound
		}
		return dp[*indx].FieldAsInterface(fldPath[1:])
	case []map[string]interface{}:
		if len(dp) <= *indx {
			return nil, ErrNotFound

		}
		return NavigableMap(dp[*indx]).FieldAsInterface(fldPath[1:])
	case []interface{}:
		if len(dp) <= *indx {
			return nil, ErrNotFound
		}
		switch ds := dp[*indx].(type) {
		case DataProvider:
			return ds.FieldAsInterface(fldPath[1:])
		case map[string]interface{}:
			return NavigableMap(ds).FieldAsInterface(fldPath[1:])
		default:
		}
	default:

	}
	err = ErrNotFound // xml compatible
	val = nil
	return
}

// FieldAsString returns the value from path as string
func (ms NavigableMap) FieldAsString(fldPath []string) (str string, err error) {
	var val interface{}
	if val, err = ms.FieldAsInterface(fldPath); err != nil {
		return
	}
	return IfaceAsString(val), nil
}

// Set sets the value at the given path
func (ms NavigableMap) Set(fldPath []string, val interface{}) (err error) {
	if len(fldPath) == 0 {
		return fmt.Errorf("Wrong path")
	}
	if len(fldPath) == 1 {
		ms[fldPath[0]] = val

		return
	}

	if _, has := ms[fldPath[0]]; !has {
		nMap := NavigableMap{}
		ms[fldPath[0]] = nMap
		return nMap.Set(fldPath[1:], val)
	}
	switch dp := ms[fldPath[0]].(type) {
	case dataStorage:
		return dp.Set(fldPath[1:], val)
	case map[string]interface{}:
		return NavigableMap(dp).Set(fldPath[1:], val)
	default:

		return fmt.Errorf("Wrong path")
	}

}

// GetKeys returns all the keys from map
func (ms NavigableMap) GetKeys(nesteed bool) (keys []string) {
	if !nesteed {
		keys = make([]string, len(ms))
		i := 0
		for k := range ms {
			keys[i] = k
			i++

		}
		return
	}
	for k, v := range ms {
		keys = append(keys, k)
		switch rv := v.(type) {
		case dataStorage:
			for _, dsKey := range rv.GetKeys(nesteed) {
				keys = append(keys, k+NestingSep+dsKey)
			}
		case map[string]interface{}:
			for _, dsKey := range NavigableMap(rv).GetKeys(nesteed) {
				keys = append(keys, k+NestingSep+dsKey)
			}
		case []NavigableMap:
			for i, dp := range rv {
				pref := k + fmt.Sprintf("[%v]", i)
				keys = append(keys, pref)
				for _, dsKey := range dp.GetKeys(nesteed) {
					keys = append(keys, pref+NestingSep+dsKey)
				}
			}
		case []dataStorage:
			for i, dp := range rv {
				pref := k + fmt.Sprintf("[%v]", i)
				keys = append(keys, pref)
				for _, dsKey := range dp.GetKeys(nesteed) {
					keys = append(keys, pref+NestingSep+dsKey)
				}
			}
		case []map[string]interface{}:
			for i, dp := range rv {
				pref := k + fmt.Sprintf("[%v]", i)
				keys = append(keys, pref)
				for _, dsKey := range NavigableMap(dp).GetKeys(nesteed) {
					keys = append(keys, pref+NestingSep+dsKey)
				}
			}
		case []interface{}:
			for i := range rv {
				keys = append(keys, k+fmt.Sprintf("[%v]", i))
			}
		case []string:
			for i := range rv {
				keys = append(keys, k+fmt.Sprintf("[%v]", i))
			}
		default:
			// ToDo:should not be called
			keys = append(keys, getPathFromInterface(v, k+NestingSep)...)
		}

	}
	return

}

// Remove removes the item at path
func (ms NavigableMap) Remove(fldPath []string) (err error) {
	if len(fldPath) == 0 {
		return fmt.Errorf("Wrong path")
	}
	var val interface{}
	var has bool
	if val, has = ms[fldPath[0]]; !has {
		return // ignore (already removed)
	}
	if len(fldPath) == 1 {
		delete(ms, fldPath[0])

		return
	}
	switch dp := val.(type) {
	case dataStorage:
		return dp.Remove(fldPath[1:])
	case map[string]interface{}:
		return NavigableMap(dp).Remove(fldPath[1:])
	default:
		return fmt.Errorf("Wrong path")

	}

}

// RemoteHost is part of dataStorage interface
func (ms NavigableMap) RemoteHost() net.Addr {
	return LocalAddr()
}

// ToDo: remove the following functions
func getPathFromValue(in reflect.Value, prefix string) (out []string) {
	switch in.Kind() {
	case reflect.Ptr:
		return getPathFromValue(in.Elem(), prefix)
	case reflect.Array, reflect.Slice:
		prefix = strings.TrimSuffix(prefix, NestingSep)
		for i := 0; i < in.Len(); i++ {
			pref := fmt.Sprintf("%s[%v]", prefix, i)
			out = append(out, pref)
			out = append(out, getPathFromValue(in.Index(i), pref+NestingSep)...)
		}
	case reflect.Map:
		iter := reflect.ValueOf(in).MapRange()
		for iter.Next() {
			pref := prefix + iter.Key().String()
			out = append(out, pref)
			out = append(out, getPathFromValue(iter.Value(), pref+NestingSep)...)
		}
	case reflect.Struct:
		inType := in.Type()
		for i := 0; i < in.NumField(); i++ {
			pref := prefix + inType.Field(i).Name
			out = append(out, pref)
			out = append(out, getPathFromValue(in.Field(i), pref+NestingSep)...)
		}
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String, reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Interface:
	default:
	}
	return
}

// used by NavigableMap2 GetKeys to return all values
func getPathFromInterface(in interface{}, prefix string) (out []string) {
	switch vin := in.(type) {
	case map[string]interface{}:
		for k, val := range vin {
			pref := prefix + k
			out = append(out, pref)
			out = append(out, getPathFromInterface(val, pref+NestingSep)...)
		}
	case []map[string]interface{}:
		prefix = strings.TrimSuffix(prefix, NestingSep)
		for i, val := range vin {
			pref := fmt.Sprintf("%s[%v]", prefix, i)
			out = append(out, pref)
			out = append(out, getPathFromInterface(val, pref+NestingSep)...)
		}
	case []interface{}:
		prefix = strings.TrimSuffix(prefix, NestingSep)
		for i, val := range vin {
			pref := fmt.Sprintf("%s[%v]", prefix, i)
			out = append(out, pref)
			out = append(out, getPathFromInterface(val, pref+NestingSep)...)
		}
	case []string:
		prefix = strings.TrimSuffix(prefix, NestingSep)
		for i := range vin {
			pref := fmt.Sprintf("%s[%v]", prefix, i)
			out = append(out, pref)
		}
	case nil, int, int32, int64, uint32, uint64, bool, float32, float64, []uint8, time.Duration, time.Time, string: //no path
	default: //reflect based
		out = getPathFromValue(reflect.ValueOf(vin), prefix)
	}
	return
}