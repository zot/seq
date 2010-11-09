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
	Find(f func(i El)bool) El
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
	return SMap(s, func(el El)El{return el})
}

func FirstN(s Seq, n int) []interface{} {
	r := make([]interface{}, n)
	x := 0
	Find(s, func(el El)bool{
		r[x] = el
		x++
		return x == n
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
	s.Find(func(el El)bool{
		result = el
		return true
	})
	return result
}

func IsEmpty(s Seq) bool {
	empty := true
	s.Find(func(el El)bool{
		empty = false
		return true
	})
	return empty
}

func Find(s Seq, f func(el El) bool) El {return s.Find(f)}

func While(s Seq, f func(el El) bool) {s.Find(func(el El)bool{return !f(el)})}

func Do(s Seq, f func(el El)) {
	s.Find(func(el El)bool{
		f(el)
		return false
	})
}

// CDo -- do f concurrently on each element of s, in any order
func CDo(s Seq, f func(el El)) {
	c := CMap(s, func(el El)El{f(el); return nil})()
	for <- c; !closed(c); <- c {}
}

func Len(s Seq) int {return s.Len()}

func Output(s Seq, c SeqChan) {
	Do(s, func(el El){c <- el})
}

func Rest(s Seq) Seq {return s.Rest()}

func Append(s1 Seq, s2 Seq) Seq {return s1.Append(s2)}

func AppendToVector(s Seq, vec *vector.Vector) {
	switch arg := s.(type) {
	case *SequentialSeq: vec.AppendVector((*vector.Vector)(arg))
	default: Do(s, func(el El){vec.Push(el)})
	}
}

func SAppend(s Seq, s2 Seq) *SequentialSeq {
	vec := make(vector.Vector, 0, quickLen(s, 8) + quickLen(s2, 8))
	AppendToVector(s, &vec)
	AppendToVector(s2, &vec)
	return (*SequentialSeq)(&vec)
}

func CAppend(s Seq, s2 Seq) ConcurrentSeq {
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

func ifFunc(condition func(e El)bool, op func(e El)) func(el El){return func(el El){if condition(el) {op(el)}}}

func SFilter(s Seq, filter func(e El)bool) *SequentialSeq {
	//continue shrinking
	vec := make(vector.Vector, 0, quickLen(s, 8))
	Do(s, ifFunc(filter, func(el El){vec.Push(el)}))
	return (*SequentialSeq)(&vec)
}

func CFilter(s Seq, filter func(e El)bool) ConcurrentSeq {
	return Gen(func(c SeqChan){
		Do(s, ifFunc(filter, func(el El){c <- el}))
	})
}

func Map(s Seq, f func(el El) El) Seq {return s.Map(f)}

func SMap(s Seq, f func(i El)El) *SequentialSeq {
	vec := make(vector.Vector, 0, quickLen(s, 8))
	Do(s, func(el El){vec.Push(f(el))})
	return (*SequentialSeq)(&vec)
}

type reply struct {
	index int;
	result El
}

type swEntry struct {
	value El
	present bool
}

// SlidingWindow is a vector with limited capacity (power of 2) and a base
type SlidingWindow struct {
	start, base, count, mask int
	values []swEntry
}
// NewSlidingWindow creates a new SlidingWindow with capacity size
func NewSlidingWindow(sz uint) *SlidingWindow {return &SlidingWindow{0, 0, 0, (1 << sz) - 1, make([]swEntry, 1 << sz)}}
func (r *SlidingWindow) Max() int {return r.base + r.Capacity()}
func (r *SlidingWindow) Capacity() int {return len(r.values)}
func (r *SlidingWindow) normalize(index int) int {return (index + r.Capacity()) & r.mask}
func (r *SlidingWindow) IsEmpty() bool {return r.count == 0}
func (r *SlidingWindow) IsFull() bool {return r.count == r.Capacity()}
func (r *SlidingWindow) GetFirst() (interface{}, bool) {return r.values[r.start].value, r.values[r.start].present}
func (r *SlidingWindow) RemoveFirst() (interface{}, bool) {
	result := r.values[r.start]
	if !result.present {return nil, false}
	r.values[r.start] = swEntry{nil, false}
	r.count--
	r.start = r.normalize(r.start + 1)
	r.base++
	return result.value, true
}
func (r *SlidingWindow) Get(index int) (interface{}, bool) {
	index -= r.base
	if index < 0 || index >= r.Capacity() {return nil, false}
	index = r.normalize(index + r.start)
	value := r.values[index]
	return value.value, value.present
}
func (r *SlidingWindow) Set(index int, value interface{}) bool {
	index -= r.base
	if index < 0 || index >= r.Capacity() {return false}
	index = r.normalize(index + r.start)
	r.values[index].value = value
	if !r.values[index].present {
		r.values[index].present = true
		r.count++
	}
	return true
}

// spawn a goroutine that does the following for each value, with up to size pending at a time:
//   spawn a goroutine to apply f to the value and send the result back in a channel
// send the results in order to the ouput channel as they are completed
func CMap(s Seq, f func(el El) El, sizePowerOpt... uint) ConcurrentSeq {
	sizePower := uint(6)
	if len(sizePowerOpt) > 0 {sizePower = sizePowerOpt[0]}
	size := 1 << sizePower
	return Gen(func(output SeqChan){
		//punt and convert sequence to concurrent
		//maybe someday we'll handle SequentialSequences separately
		input := Concurrent(s)()
		window := NewSlidingWindow(sizePower)
		replyChannel := make(chan reply)
		inputCount, pendingInput, pendingOutput := 0, 0, 0
		inputClosed := false
		defer close(replyChannel)
		for !inputClosed || pendingInput > 0 || pendingOutput > 0 {
			first, hasFirst := window.GetFirst()
			ic, oc, rc := input, output, replyChannel
			if !hasFirst {oc = nil}
			if inputClosed || pendingInput >= size {ic = nil}
			if pendingOutput >= size {rc = nil}
			select {
			case oc <- first:
				window.RemoveFirst()
				pendingOutput--
			case inputElement := <- ic:
				if closed(ic) {
					inputClosed = true
				} else {
					go func(index int, value interface{}) {
						replyChannel <- reply{index, f(value)}
					}(inputCount, inputElement)
					inputCount++
					pendingInput++
				}
			case replyElement := <- rc:
				window.Set(replyElement.index, replyElement.result)
				pendingInput--
				pendingOutput++
			}
		}
	})
}

func FlatMap(s Seq, f func(el El) Seq) Seq {return s.FlatMap(f)}

func SFlatMap(s Seq, f func(i El) Seq) *SequentialSeq {
	vec := make(vector.Vector, 0, quickLen(s, 8))
	Do(s, func(e El){Do(f(e).(Seq), func(sub El){vec.Push(sub)})})
	return (*SequentialSeq)(&vec)
}

func CFlatMap(s Seq, f func(i El) Seq, sizeOpt... uint) ConcurrentSeq {
	return Gen(func(c SeqChan){
		Do(CMap(s, func(e El)El{return f(e)}, sizeOpt...), func(sub El){
			Output(sub.(Seq), c)
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

//var Names map[interface{}]string

//returns the product of the Seqs contained in sequences
func Product(sequences Seq) Seq {
	return Fold(sequences, From(From()), func(result, each El)El{
		return result.(Seq).FlatMap(func(seq El)Seq{
			return each.(Seq).Map(func(i El) El {
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

//This pretty is ugly :)
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

func (s ConcurrentSeq) Find(f func(el El)bool) El {
	c := s()
	defer close(c)
	for el := <- c; !closed(c) ; el = <- c {
		if f(el) {return el}
	}
	return nil
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

// recursively convert nested concurrent sequences to sequential
// does not descend into nested sequential sequences
func (s ConcurrentSeq) ToSequentialSeq() *SequentialSeq {
	return SMap(s, func(el El)El{
		switch seq := el.(type) {case ConcurrentSeq: return seq.ToSequentialSeq()}
		return el
	})
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

func (s *SequentialSeq) Find(f func(el El)bool) El {
	for i := 0; i < len(*s); i++ {
		if f((*s)[i]) {return (*s)[i]}
	}
	return nil
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
