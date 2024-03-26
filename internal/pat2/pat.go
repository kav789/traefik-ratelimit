package pat

import (
	"fmt"
//	"slices"
	"strconv"
	"strings"
)

const (
	TYPE_VAL = iota
	TYPE_ANY
	TYPE_LEN
)

type Pat struct {
	t int8
	v int8
}

func slicesEqual(s1, s2 []Pat) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}


func Appendpat(pats [][]Pat, p []Pat) [][]Pat {
	if p == nil {
		return pats
	}
	for _, tp := range pats {
//		if slices.Equal(tp, p) {
		if slicesEqual(tp, p) {
			return pats
		}
	}
	return append(pats, p)
}

func Preparepat(pt []Pat, s string) (string, bool) {
	ss := strings.Split(s, "/")
	r := make([]string, 0, len(pt))
ptloop:
	for _, p := range pt {
		v := ""
		i := int(p.v)
		switch p.t {
		case TYPE_VAL:
		case TYPE_ANY:
			v = "*"
		case TYPE_LEN:
			if i != len(ss)-1 {
				return "", false
			}
			r = append(r, strconv.Itoa(i)+":$")
			continue ptloop
		}
		j := i
		if i < 0 {
			j = len(ss) + i
		}
		if j > len(ss)-1 || j < 1 {
			return "", false
		}
		if len(v) == 0 {
			v = ss[j]
		}
		r = append(r, strconv.Itoa(i)+":"+v)
	}
	return strings.Join(r, "/"), true
}

func Compilepat(s string) (string, []Pat, error) {
	if len(strings.TrimSpace(s)) == 0 {
		return "", nil, nil
	}
	eo := false
	if strings.HasSuffix(s, "$") {
		eo = true
		s = s[:len(s)-1]
	}
	ss := strings.Split(s, "/")
	r := make([]string, 0, len(ss))
	ri := make([]Pat, 0, len(ss))
	f, fl, ril := int8(0), false, int8(0)
	for i, s := range ss {
		switch s {
		case "**":
			if fl || eo {
				return "", nil, fmt.Errorf("bad pattern")
			}
			fl = true
			f = int8(i) + 1
		case "*":
			r = append(r, s)
			ril = int8(i)
			ri = append(ri, Pat{
				t: TYPE_ANY,
				v: ril,
			})
		default:
			if len(s) == 0 && i == 0 {
				break
			}
			r = append(r, s)
			ril = int8(i)
			ri = append(ri, Pat{
				t: TYPE_VAL,
				v: ril,
			})
		}
	}
	if eo {
		r = append(r, "$")
		ri = append(ri, Pat{
			t: TYPE_LEN,
			v: int8(len(ss)) - 1,
		})
	}
	for i := range r {
		v := 0
		switch ri[i].t {
		case TYPE_VAL, TYPE_ANY:
			if ri[i].v >= f && fl {
				ri[i].v = ri[i].v - ril - 1
			}
			v = int(ri[i].v)
		case TYPE_LEN:
			v = int(ri[i].v)
		}
		r[i] = strconv.Itoa(v) + ":" + r[i]
	}
	return strings.Join(r, "/"), ri, nil
}
