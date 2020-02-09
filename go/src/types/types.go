package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Errors/Exceptions
type MalError struct {
	Obj MalType
}

func (e MalError) Error() string {
	return fmt.Sprintf("%#v", e.Obj)
}

// General types
type MalType interface {
}

type EnvType interface {
	Find(key Symbol) EnvType
	Set(key Symbol, value MalType) MalType
	Get(key Symbol) (MalType, error)
}

// Scalars
func Nil_Q(obj MalType) bool {
	return obj == nil
}

func True_Q(obj MalType) bool {
	b, ok := obj.(bool)
	return ok && b == true
}

func False_Q(obj MalType) bool {
	b, ok := obj.(bool)
	return ok && b == false
}

func Number_Q(obj MalType) bool {
	_, ok := obj.(int)
	return ok
}

// Symbols
type Symbol struct {
	Val string `json:"symbol"`
}

func Symbol_Q(obj MalType) bool {
	_, ok := obj.(Symbol)
	return ok
}

// Keywords
func NewKeyword(s string) (MalType, error) {
	return "\u029e" + s, nil
}

func Keyword_Q(obj MalType) bool {
	s, ok := obj.(string)
	return ok && strings.HasPrefix(s, "\u029e")
}

// Strings
func String_Q(obj MalType) bool {
	_, ok := obj.(string)
	return ok
}

// Functions
type Func struct {
	Fn   func([]MalType) (MalType, error) `json:"fn"`
	Meta MalType                          `json:"meta,omitempty"`
}

func Func_Q(obj MalType) bool {
	_, ok := obj.(Func)
	return ok
}

type MalFunc struct {
	Eval    func(MalType, EnvType) (MalType, error)
	Exp     MalType
	Env     EnvType
	Params  MalType
	IsMacro bool
	GenEnv  func(EnvType, MalType, MalType) (EnvType, error)
	Meta    MalType
}

func MalFunc_Q(obj MalType) bool {
	_, ok := obj.(MalFunc)
	return ok
}

func (f MalFunc) SetMacro() MalType {
	f.IsMacro = true
	return f
}

func (f MalFunc) GetMacro() bool {
	return f.IsMacro
}

// Take either a MalFunc or regular function and apply it to the
// arguments
func Apply(f_mt MalType, a []MalType) (MalType, error) {
	switch f := f_mt.(type) {
	case MalFunc:
		env, e := f.GenEnv(f.Env, f.Params, List{a, nil})
		if e != nil {
			return nil, e
		}
		return f.Eval(f.Exp, env)
	case Func:
		return f.Fn(a)
	case func([]MalType) (MalType, error):
		return f(a)
	default:
		return nil, errors.New("Invalid function to Apply")
	}
}

// Lists
type List struct {
	Val  ListMalType `json:"list"`
	Meta MalType     `json:"meta,omitempty"`
}

type ListMalType []MalType

// UnmarshalJSON custom unmarshaller for J
func (j *ListMalType) UnmarshalJSON(b []byte) (err error) {
	rawJSONAST := []json.RawMessage{}
	err = JSONUnmarshal(b, &rawJSONAST)
	if err != nil {
		return err
	}

	for _, raw := range rawJSONAST {
		switch raw[0] {
		case '{':
			m := map[string]interface{}{}
			err = json.Unmarshal(raw, &m)
			if err != nil {
				return err
			}
			if _, ok := m["symbol"]; ok {
				res := Symbol{}
				if e := json.Unmarshal(raw, &res); e != nil {
					return fmt.Errorf("json-decode of symbol failed: %v", e)
				}
				*j = append(*j, res)
			} else if _, ok := m["atom"]; ok {
				res := Atom{}
				if e := json.Unmarshal(raw, &res); e != nil {
					return fmt.Errorf("json-decode of atom failed: %v", e)
				}
				*j = append(*j, res)
			} else if _, ok := m["list"]; ok {
				res := List{}
				if e := json.Unmarshal(raw, &res); e != nil {
					return fmt.Errorf("json-decode of list failed: %v", e)
				}
				*j = append(*j, res)
			} else if _, ok := m["hashmap"]; ok {
				res := HashMap{}
				if e := json.Unmarshal(raw, &res); e != nil {
					return fmt.Errorf("json-decode of hashmap failed: %v", e)
				}
				*j = append(*j, res)
			} else if _, ok := m["vector"]; ok {
				res := Vector{}
				if e := json.Unmarshal(raw, &res); e != nil {
					return fmt.Errorf("json-decode of vector failed: %v", e)
				}
				*j = append(*j, res)
			} else if _, ok := m["fn"]; ok { // won't work
				res := Func{}
				if e := json.Unmarshal(raw, &res); e != nil {
					return fmt.Errorf("json-decode of fn failed: %v", e)
				}
				*j = append(*j, res)
			} else {
				return errors.New("json-decode of unknown type")
			}
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			item := json.Number("")
			err = JSONUnmarshal(raw, &item)
			if err != nil {
				return err
			}
			*j = append(*j, item)
		default:
			var item interface{}
			err = JSONUnmarshal(raw, &item)
			if err != nil {
				return err
			}
			*j = append(*j, item)
		}
	}
	return nil
}

func JSONUnmarshal(buffer []byte, ast interface{}) error {
	reader := bytes.NewReader(buffer)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	return decoder.Decode(ast)
}

func NewList(a ...MalType) MalType {
	return List{a, nil}
}

func List_Q(obj MalType) bool {
	_, ok := obj.(List)
	return ok
}

// Vectors
type Vector struct {
	Val  []MalType `json:"vector"`
	Meta MalType   `json:"meta,omitempty"`
}

func Vector_Q(obj MalType) bool {
	_, ok := obj.(Vector)
	return ok
}

func GetSlice(seq MalType) ([]MalType, error) {
	switch obj := seq.(type) {
	case List:
		return obj.Val, nil
	case Vector:
		return obj.Val, nil
	default:
		return nil, errors.New("GetSlice called on non-sequence")
	}
}

// Hash Maps
type HashMap struct {
	Val  map[string]MalType `json:"hashmap"`
	Meta MalType            `json:"meta,omitempty"`
}

func NewHashMap(seq MalType) (MalType, error) {
	lst, e := GetSlice(seq)
	if e != nil {
		return nil, e
	}
	if len(lst)%2 == 1 {
		return nil, errors.New("Odd number of arguments to NewHashMap")
	}
	m := map[string]MalType{}
	for i := 0; i < len(lst); i += 2 {
		str, ok := lst[i].(string)
		if !ok {
			return nil, errors.New("expected hash-map key string")
		}
		m[str] = lst[i+1]
	}
	return HashMap{m, nil}, nil
}

func HashMap_Q(obj MalType) bool {
	_, ok := obj.(HashMap)
	return ok
}

// Atoms
type Atom struct {
	Val  MalType `json:"atom"`
	Meta MalType `json:"meta,omitempty"`
}

func (a *Atom) Set(val MalType) MalType {
	a.Val = val
	return a
}

func Atom_Q(obj MalType) bool {
	_, ok := obj.(*Atom)
	return ok
}

// General functions

func _obj_type(obj MalType) string {
	if obj == nil {
		return "nil"
	}
	return reflect.TypeOf(obj).Name()
}

func Sequential_Q(seq MalType) bool {
	if seq == nil {
		return false
	}
	return (reflect.TypeOf(seq).Name() == "List") ||
		(reflect.TypeOf(seq).Name() == "Vector")
}

func Equal_Q(a MalType, b MalType) bool {
	ota := reflect.TypeOf(a)
	otb := reflect.TypeOf(b)
	if !((ota == otb) || (Sequential_Q(a) && Sequential_Q(b))) {
		return false
	}
	//av := reflect.ValueOf(a); bv := reflect.ValueOf(b)
	//fmt.Printf("here2: %#v\n", reflect.TypeOf(a).Name())
	//switch reflect.TypeOf(a).Name() {
	switch a.(type) {
	case Symbol:
		return a.(Symbol).Val == b.(Symbol).Val
	case List:
		as, _ := GetSlice(a)
		bs, _ := GetSlice(b)
		if len(as) != len(bs) {
			return false
		}
		for i := 0; i < len(as); i += 1 {
			if !Equal_Q(as[i], bs[i]) {
				return false
			}
		}
		return true
	case Vector:
		as, _ := GetSlice(a)
		bs, _ := GetSlice(b)
		if len(as) != len(bs) {
			return false
		}
		for i := 0; i < len(as); i += 1 {
			if !Equal_Q(as[i], bs[i]) {
				return false
			}
		}
		return true
	case HashMap:
		am := a.(HashMap).Val
		bm := b.(HashMap).Val
		if len(am) != len(bm) {
			return false
		}
		for k, v := range am {
			if !Equal_Q(v, bm[k]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
