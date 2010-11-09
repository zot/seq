// Copyright 2010 Bill Burdick. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
package seq

import "fmt"
import "io"
import "os"
import "container/vector"

type El interface{}
type Seq interface {
	// core methods
	While(f func(i El)bool)
	Rest() Seq
	Len() int
	// wrappers that keep the current type
	Append(s2 Seq) Seq
	Prepend(s2 Seq) Seq
	Filter(filter func(e El) bool) Seq
	Map(f func(i El) El) Seq
	FlatMap(f func(i El) Seq) Seq
}

//convert a sequence to a concurrent sequence
func Concurrent(s Seq) ConcurrentSeq {
	switch seq := s.(type) {case ConcurrentSeq: return seq}
	return Gen(func(c SeqChan){Output(s, c)})
}

//convert a sequence to a sequential sequence
func Sequential(s Seq) *SequentialSeq {
	switch seq := s.(type) {case *SequentialSeq: return seq}
	vec := make(vector.Vector, 0, 8)
	Do(s, func(v El){vec.Push(v)})
	return (*SequentialSeq)(&vec)
}

func FirstN(s Seq, n int) []interface{} {
	r := make([]interface{}, n)
	x := 0
	While(s, func(el El)bool{
		r[x] = el
		x++
		return x < n
	})
	return r
}

func First2(s Seq) (a, b interface{}) {
	r := FirstN(s, 2)
	return r[0], r[1]
}

func First3(s Seq) (a, b, c interface{}) {
	r := FirstN(s, 3)
	return r[0], r[1], r[2]
}

func First4(s Seq) (a, b, c, d interface{}) {
	r := FirstN(s, 4)
	return r[0], r[1], r[2], r[3]
}

func First5(s Seq) (a, b, c, d, e interface{}) {
	r := FirstN(s, 5)
	return r[0], r[1], r[2], r[3], r[4]
}

func First6(s Seq) (a, b, c, d, e, f interface{}) {
	r := FirstN(s, 6)
	return r[0], r[1], r[2], r[3], r[4], r[5]
}

func IsSeq(s interface{}) bool {
	_, test := s.(Seq)
	return test
}

func First(s Seq) interface{} {
	var result interface{}
	s.While(func(el El)bool{
		result = el
		return false
	})
	return result
}

func IsEmpty(s Seq) bool {
	empty := true
	s.While(func(el El)bool{
		empty = false
		return false
	})
	return empty
}

func While(s Seq, f func(el El) bool) {s.While(f)}

func Do(s Seq, f func(el El)) {
	s.While(func(el El)bool{
		f(el)
		return true
	})
}

func Len(s Seq) int {return s.Len()}

func Output(s Seq, c SeqChan) {
	Do(s, func(el El){
		c <- el
	})
}

func Rest(s Seq) Seq {return s.Rest()}

func Append(s1 Seq, s2 Seq) Seq {return s1.Append(s2)}

func AppendToVector(s Seq, vec *vector.Vector) {
	switch arg := s.(type) {
	case *SequentialSeq: vec.AppendVector((*vector.Vector)(arg))
	default: Do(s, func(el El){vec.Push(el)})
	}
}

func SAppend(s Seq, s2 Seq) Seq {
	vec := make(vector.Vector, 0, quickLen(s, 8) + quickLen(s2, 8))
	AppendToVector(s, &vec)
	AppendToVector(s2, &vec)
//print("SAppend ");Prettyln(s);print(" + ");Prettyln(s2);println(" = ");Prettyln((*SequentialSeq)(&vec))
	return (*SequentialSeq)(&vec)
}

func CAppend(s Seq, s2 Seq) Seq {
	return Gen(func(c SeqChan){
		Output(s, c)
		Output(s2, c)
	})
}

func Prepend(s1 Seq, s2 Seq) Seq {return s1.Prepend(s2)}

func quickLen(s Seq, d int) int {
	switch seq := s.(type) {case *SequentialSeq: return s.Len()}
	return d
}

func Filter(s Seq, filter func(e El)bool) Seq {return s.Filter(filter)}

func SFilter(s Seq, filter func(e El)bool) Seq {
	//continue shrinking
	vec := make(vector.Vector, 0, quickLen(s, 8))
	Do(s, func(el El){
		if filter(el) {vec.Push(el)}
	})
	return (*SequentialSeq)(&vec)
}

func CFilter(s Seq, filter func(e El)bool) Seq {
	return Gen(func(c SeqChan){
		Do(s, func(el El){
			if filter(el) {c <- el}
		})
	})
}

func Map(s Seq, f func(el El) El) Seq {return s.Map(f)}

func SMap(s Seq, f func(i El)El) Seq {
	vec := make(vector.Vector, 0, quickLen(s, 8))
	Do(s, func(el El){vec.Push(f(el))})
	return (*SequentialSeq)(&vec)
}

func CMap(s Seq, f func(el El) El) Seq {
	return Gen(func(c SeqChan) {
		Do(s, func(v El){c <- f(v)})
	})
}

func FlatMap(s Seq, f func(el El) Seq) Seq {return s.FlatMap(f)}

func SFlatMap(s Seq, f func(i El) Seq) Seq {
	vec := make(vector.Vector, 0, quickLen(s, 8))
	Do(s, func(el El){
		Do(f(el), func(sub El){vec.Push(sub)})
	})
	return (*SequentialSeq)(&vec)
}

func CFlatMap(s Seq, f func(i El) Seq) Seq {
	return Gen(func(c SeqChan) {
		Do(s, func(v El){
			Do(f(v), func(sub El){c <- sub})
		})
	})
}

func Fold(s Seq, init interface{}, f func(acc, el El)El) interface{} {
	Do(s, func(el El){init = f(init, el)})
	return init
}

//maybe convert this to use an accumulator instead of append?
func Combinations(s Seq, number int) Seq {
	if number == 0 || IsEmpty(s) {return From(From())}
	return Combinations(s.Rest(), number).Prepend(Combinations(s.Rest(), number - 1).Map(func(el El)El{
		return el.(Seq).Prepend(From(First(s)))
	}))
}

//returns the product of the Seqs contained in sequences
func Product(sequences Seq) Seq {
	return Fold(sequences, From(From()), func(result, each El)El{
//fmt.Print("folding: ");Pretty(each);fmt.Print(" into ");Prettyln(result)
		return result.(Seq).FlatMap(func(seq El)Seq{
//fmt.Print("flat map with: ");Prettyln(seq)
			return each.(Seq).Map(func(i El) El {
//fmt.Print("map with: ");Prettyln(i)
				return seq.(Seq).Append(From(i))
			})
		})
	}).(Seq)
}

func Prettyln(s interface{}, rest... interface{}) {
	writer := Pretty(s, rest...)
	fmt.Fprintln(writer)
}
func Pretty(s interface{}, args... interface{}) io.Writer {
	var writer io.Writer = os.Stdout
	var names map[interface{}]string
	for i := 0; i < len(args); i++ {
		switch arg := args[i].(type) {
		case map[interface{}]string: names = arg
		case io.Writer: writer = arg
		}
	}
	if names == nil {names = map[interface{}]string{}}
	prettyLevel(s, 0, names, writer)
	return writer
}

//This is pretty ugly :)
func prettyLevel(s interface{}, level int, names map[interface{}]string, w io.Writer) {
	name, hasName := names[s]
	if hasName {
		fmt.Fprint(w, name)
	} else switch arg := s.(type) {
	case Seq:
		fmt.Fprintf(w, "%*s%s", level, "", "[")
		first := true
		innerSeq := false
		named := false
		Do(arg, func(v El) {
			_, named = names[v]
			_,innerSeq = v.(Seq)
			if first {
				first = false
				if !named && innerSeq {
					fmt.Fprintln(w)
				}
			} else if !named && innerSeq {
				fmt.Fprintln(w, ",")
			} else {
				fmt.Fprint(w, ", ")
			}
			if innerSeq {
				prettyLevel(v.(Seq), level + 4, names, w)
			} else {
				fmt.Fprintf(w, "%v", v)
			}
		})
		if innerSeq {
			if !named {
				fmt.Fprintf(w, "\n%*s", level, "")
			}
		}
		fmt.Fprintf(w, "]")
	default:
		fmt.Print(arg)
	}
}

//ConcurrentSeq

type SeqChan chan interface{}

type ConcurrentSeq func()SeqChan

// f must behave properly when the channel is closed, so that IsEmpty and First work properly
func Gen(f func(c SeqChan)) ConcurrentSeq {
	return func() SeqChan {
		c := make(SeqChan)
		go func() {
			defer close(c)
			f(c)
		}()
		return c
	}
}

func CUpto(limit int) ConcurrentSeq {
	return Gen(func(c SeqChan) {
		for i := 0; i < limit; i++ {
			c <- i
		}
	})
}

func (s ConcurrentSeq) While(f func(el El)bool) {
	c := s()
	defer close(c)
	for el := <- c; !closed(c) && f(el); el = <- c {}
}

func (s ConcurrentSeq) Rest() Seq {
	return ConcurrentSeq(func()SeqChan{
		c := s()
		<- c
		return c
	})
}

func (s ConcurrentSeq) Len() int {
	len := 0
	Do(s, func(el El){
		len++
	})
	return len
}

func (s ConcurrentSeq) Append(s2 Seq) Seq {return CAppend(s, s2)}

func (s ConcurrentSeq) Prepend(s2 Seq) Seq {return CAppend(s2, s)}

func (s ConcurrentSeq) Filter(f func(e El)bool) Seq {return CFilter(s, f)}

func (s ConcurrentSeq) Map(f func(i El)El) Seq {return CMap(s, f)}

func (s ConcurrentSeq) FlatMap(f func(i El) Seq) Seq {return CFlatMap(s, f)}

func toSequentialSeq(el interface{}) interface{} {
	switch seq := el.(type) {
	case ConcurrentSeq: return seq.ToSequentialSeq()
	case []interface{}:
		cpy := make([]interface{}, len(seq))
		copy(cpy, seq)
		return cpy
	}
	return el
}

func (s ConcurrentSeq) ToSequentialSeq() *SequentialSeq {
	vec := make(vector.Vector, 0, 8)
	Do(s, func(v El){vec.Push(toSequentialSeq(v))})
	return (*SequentialSeq)(&vec)
}


// SequentialSeq

type SequentialSeq []interface{}

func From(els... interface{}) *SequentialSeq {return (*SequentialSeq)(&els)}

func AUpto(limit int) *SequentialSeq {
	a := make([]interface{}, limit)
	for i := 0; i < limit; i++ {
		a[i] = i
	}
	return (*SequentialSeq)(&a)
}

func (s *SequentialSeq) While(f func(el El)bool) {
	for i := 0; i < len(*s) && f((*s)[i]); i++ {}
}

func (s *SequentialSeq) Rest() Seq {
	s2 := (*s)[1:]
	return (*SequentialSeq)(&s2)
}

func (s *SequentialSeq) Len() int {return len(*s)}

func (s *SequentialSeq) Append(s2 Seq) Seq {return SAppend(s, s2)}

func (s *SequentialSeq) Prepend(s2 Seq) Seq {return SAppend(s2, s)}

func (s *SequentialSeq) Filter(f func(e El)bool) Seq {return SFilter(s, f)}

func (s *SequentialSeq) Map(f func(i El)El) Seq {return SMap(s, f)}

func (s *SequentialSeq) FlatMap(f func(i El) Seq) Seq {return SFlatMap(s, f)}
