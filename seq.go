// Copyright 2010 Bill Burdick. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
package seq

import "fmt"
import "io"
import "os"
import "container/vector"

// convenience alias for sequence elements
type El interface{}
// basic sequence support
type Seq interface {
	Find(f func(i El)bool) El
	Rest() Seq
	Len() int
	Append(s2 Seq) Seq
	Prepend(s2 Seq) Seq
	Filter(filter func(e El) bool) Seq
	Map(f func(i El) El) Seq
	FlatMap(f func(i El) Seq) Seq
}

//convert a sequence to a concurrent sequence (if necessary)
func Concurrent(s Seq) ConcurrentSeq {
	switch seq := s.(type) {case ConcurrentSeq: return seq}
	return Gen(func(c SeqChan){Output(s, c)})
}

//convert a sequence to a sequential sequence (if necessary)
func Sequential(s Seq) *SequentialSeq {
	switch seq := s.(type) {case *SequentialSeq: return seq}
	return SMap(s, func(el El)El{return el})
}

//returns a new array of the first N items
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

//convenience function with multiple return values
func First2(s Seq) (a, b interface{}) {
	r := FirstN(s, 2)
	return r[0], r[1]
}

//convenience function with multiple return values
func First3(s Seq) (a, b, c interface{}) {
	r := FirstN(s, 3)
	return r[0], r[1], r[2]
}

//convenience function with multiple return values
func First4(s Seq) (a, b, c, d interface{}) {
	r := FirstN(s, 4)
	return r[0], r[1], r[2], r[3]
}

//convenience function with multiple return values
func First5(s Seq) (a, b, c, d, e interface{}) {
	r := FirstN(s, 5)
	return r[0], r[1], r[2], r[3], r[4]
}

//convenience function with multiple return values
func First6(s Seq) (a, b, c, d, e, f interface{}) {
	r := FirstN(s, 6)
	return r[0], r[1], r[2], r[3], r[4], r[5]
}

//returns whether s can be interpreted as a sequence
func IsSeq(s interface{}) bool {
	_, test := s.(Seq)
	return test
}

//returns the first item in a sequence
func First(s Seq) interface{} {
	var result interface{}
	s.Find(func(el El)bool{
		result = el
		return true
	})
	return result
}

//returns whether a sequence is empty
func IsEmpty(s Seq) bool {
	empty := true
	s.Find(func(el El)bool{
		empty = false
		return true
	})
	return empty
}

//returns the first item in a sequence for which f returns true or nil if none is found
func Find(s Seq, f func(el El) bool) El {return s.Find(f)}

//applies f to each item in the sequence until f returns false
func While(s Seq, f func(el El) bool) {s.Find(func(el El)bool{return !f(el)})}

//applies f to each item in the sequence
func Do(s Seq, f func(el El)) {
	s.Find(func(el El)bool{
		f(el)
		return false
	})
}

//applies f concurrently to each element of s, in no particular order; sizePowerOpt will default to {6} and CMap will allow up to 1 << sizePowerOpt[0] outstanding concurrent instances of f at any time
func CDo(s Seq, f func(el El), sizePowerOpt... uint) {
	c := CMap(s, func(el El)El{f(el); return nil}, sizePowerOpt...)()
	for <- c; !closed(c); <- c {}
}

//returns the length of s
func Len(s Seq) int {return s.Len()}

//sends each item of s to c
func Output(s Seq, c SeqChan) {Do(s, func(el El){c <- el})}

//returns a new sequence of the same type as s consisting of all of the elements of s except for the first one
func Rest(s Seq) Seq {return s.Rest()}

//returns a new sequence of the same type as s1 that appends this s1 and s2
func Append(s1 Seq, s2 Seq) Seq {return s1.Append(s2)}

//append a sequence to a vector
func AppendToVector(vec *vector.Vector, s Seq) {
	switch arg := s.(type) {
	case *SequentialSeq: vec.AppendVector((*vector.Vector)(arg))
	default: Do(s, func(el El){vec.Push(el)})
	}
}

//returns a new SequentialSeq which consists of appending s and s2
func SAppend(s Seq, s2 Seq) *SequentialSeq {
	vec := make(vector.Vector, 0, quickLen(s, 8) + quickLen(s2, 8))
	AppendToVector(&vec, s)
	AppendToVector(&vec, s2)
	return (*SequentialSeq)(&vec)
}

//returns a new ConcurrentSeq which consists of appending s and s2
func CAppend(s Seq, s2 Seq) ConcurrentSeq {
	return Gen(func(c SeqChan){
		Output(s, c)
		Output(s2, c)
	})
}

func quickLen(s Seq, d int) int {
	switch seq := s.(type) {case *SequentialSeq: return s.Len()}
	return d
}

//returns a new sequence of the same type as s consisting of the elements of s for which filter returns true
func Filter(s Seq, filter func(e El)bool) Seq {return s.Filter(filter)}

func ifFunc(condition func(e El)bool, op func(e El)) func(el El){return func(el El){if condition(el) {op(el)}}}

//returns a new SequentialSeq consisting of the elements of s for which filter returns true
func SFilter(s Seq, filter func(e El)bool) *SequentialSeq {
	//continue shrinking
	vec := make(vector.Vector, 0, quickLen(s, 8))
	Do(s, ifFunc(filter, func(el El){vec.Push(el)}))
	return (*SequentialSeq)(&vec)
}

//returns a new ConcurrentSeq consisting of the elements of s for which filter returns true; sizePowerOpt will default to {6} and CMap will allow up to 1 << sizePowerOpt[0] outstanding concurrent instances of f at any time
func CFilter(s Seq, filter func(e El)bool, sizePowerOpt... uint) ConcurrentSeq {
	return Gen(func(c SeqChan){
		CDo(s, ifFunc(filter, func(el El){c <- el}), sizePowerOpt...)
	})
}

//returns a new sequence of the same type as s consisting of the results of appying f to the elements of s
func Map(s Seq, f func(el El) El) Seq {return s.Map(f)}

//returns a new SequentialSeq consisting of the results of appying f to the elements of s
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

// a vector with limited capacity (power of 2) and a base
type SlidingWindow struct {
	start, base, count, mask int
	values []swEntry
}
//creates a new SlidingWindow with capacity size
func NewSlidingWindow(sz uint) *SlidingWindow {return &SlidingWindow{0, 0, 0, (1 << sz) - 1, make([]swEntry, 1 << sz)}}
//returns the current maximum available index
func (r *SlidingWindow) Max() int {return r.base + r.Capacity() - 1}
//returns the size of the window
func (r *SlidingWindow) Capacity() int {return len(r.values)}
//returns the number of items in the window
func (r *SlidingWindow) Count() int {return r.count}
func (r *SlidingWindow) normalize(index int) int {return (index + r.Capacity()) & r.mask}
//returns whether the window is empty
func (r *SlidingWindow) IsEmpty() bool {return r.count == 0}
//returns whether the window has any available space
func (r *SlidingWindow) IsFull() bool {return r.count == r.Capacity()}
//returns the first item, or nil if there is none, and also returns whether there was an item
func (r *SlidingWindow) GetFirst() (interface{}, bool) {return r.values[r.start].value, r.values[r.start].present}
//removes the first item, if there is one, and also returns whether an item was removed
func (r *SlidingWindow) RemoveFirst() (interface{}, bool) {
	result := r.values[r.start]
	if !result.present {return nil, false}
	r.values[r.start] = swEntry{nil, false}
	r.count--
	r.start = r.normalize(r.start + 1)
	r.base++
	return result.value, true
}
//returns item at index, if there is one, and also returns whether an item was there
func (r *SlidingWindow) Get(index int) (interface{}, bool) {
	index -= r.base
	if index < 0 || index >= r.Capacity() {return nil, false}
	index = r.normalize(index + r.start)
	value := r.values[index]
	return value.value, value.present
}
//sets the item at index to value, if the space is available, and also returns whether an item was set
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

//returns a new ConcurrentSeq consisting of the results of appying f to the elements of s; sizePowerOpt will default to {6} and CMap will allow up to 1 << sizePowerOpt[0] outstanding concurrent instances of f at any time
func CMap(s Seq, f func(el El) El, sizePowerOpt... uint) ConcurrentSeq {
// spawn a goroutine that does the following for each value, with up to size pending at a time:
//   spawn a goroutine to apply f to the value and send the result back in a channel
// send the results in order to the ouput channel as they are completed
	sizePower := uint(6)
	if len(sizePowerOpt) > 0 {sizePower = sizePowerOpt[0]}
	size := 1 << sizePower
	return Gen(func(output SeqChan){
		//punt and convert sequence to concurrent
		//maybe someday we'll handle SequentialSequences separately
		input := Concurrent(s)()
		window := NewSlidingWindow(sizePower)
		replyChannel := make(chan reply)
		inputCount, pendingInput := 0, 0
		inputClosed := false
		defer close(replyChannel)
		for !inputClosed || pendingInput > 0 || window.Count() > 0 {
			first, hasFirst := window.GetFirst()
			ic, oc, rc := input, output, replyChannel
			if !hasFirst {oc = nil}
			if inputClosed || pendingInput >= size {ic = nil}
			if window.Count() >= size {rc = nil}
			select {
			case oc <- first: window.RemoveFirst()
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
			}
		}
	})
}

//returns a new sequence of the same type as s consisting of the concatenation of the sequences f returns when applied to all of the elements of s
func FlatMap(s Seq, f func(el El) Seq) Seq {return s.FlatMap(f)}

//returns a new SequentialSeq consisting of the concatenation of the sequences f returns when applied to all of the elements of s
func SFlatMap(s Seq, f func(i El) Seq) *SequentialSeq {
	vec := make(vector.Vector, 0, quickLen(s, 8))
	Do(s, func(e El){Do(f(e).(Seq), func(sub El){vec.Push(sub)})})
	return (*SequentialSeq)(&vec)
}

//returns a new ConcurrentSeq consisting of the concatenation of the sequences f returns when applied to all of the elements of s; sizePowerOpt will default to {6} and CMap will allow up to 1 << sizePowerOpt[0] outstanding concurrent instances of f at any time
func CFlatMap(s Seq, f func(i El) Seq, sizePowerOpt... uint) ConcurrentSeq {
	return Gen(func(c SeqChan){
		Do(CMap(s, func(e El)El{return f(e)}, sizePowerOpt...), func(sub El){
			Output(sub.(Seq), c)
		})
	})
}

//returns the result of applying f to its previous value and each element of s in succession, starting with init as the initial "previous value" for f
func Fold(s Seq, init interface{}, f func(acc, el El)El) interface{} {
	Do(s, func(el El){init = f(init, el)})
	return init
}

//returns a new sequence of the same type as s consisting of all possible combinations of the elements of s of size number or smaller
func Combinations(s Seq, number int) Seq {
	if number == 0 || IsEmpty(s) {return From(From())}
	return Combinations(s.Rest(), number).Prepend(Combinations(s.Rest(), number - 1).Map(func(el El)El{
		return el.(Seq).Prepend(From(First(s)))
	}))
}

//returns the product of the elements of sequences, where each element is a sequence
func Product(sequences Seq) Seq {
	return Fold(sequences, From(From()), func(result, each El)El{
		return result.(Seq).FlatMap(func(seq El)Seq{
			return each.(Seq).Map(func(i El) El {
				return seq.(Seq).Append(From(i))
			})
		})
	}).(Seq)
}

//pretty print an object, followed by a newline.  Optional arguments are a map of names (map[interface{}]string) and an io.Writer to write output to
func Prettyln(s interface{}, rest... interface{}) {
	writer := Pretty(s, rest...)
	fmt.Fprintln(writer)
}
//pretty print an object.  Optional arguments are a map of names (map[interface{}]string) and an io.Writer to write output to
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


//a channel which can transport sequence elements
type SeqChan chan interface{}

//A concurrent sequence.  You can call it to get a channel on a new goroutine, but you must make sure you read all of the items from the channel or else close it
type ConcurrentSeq func()SeqChan

//returns a new ConcurrentSeq which consists of all of the items that f writes to the channel
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

//returns a new ConcurrentSeq consisting of the numbers from 0 to limit, in succession
func CUpto(limit int) ConcurrentSeq {
	return Gen(func(c SeqChan) {
		for i := 0; i < limit; i++ {
			c <- i
		}
	})
}

//returns the first item in a sequence for which f returns true or nil if none is found
func (s ConcurrentSeq) Find(f func(el El)bool) El {
	c := s()
	defer close(c)
	for el := <- c; !closed(c) ; el = <- c {
		if f(el) {return el}
	}
	return nil
}

//returns a new ConcurrentSeq consisting of all of the elements of s except for the first one
func (s ConcurrentSeq) Rest() Seq {
	return ConcurrentSeq(func()SeqChan{
		c := s()
		<- c
		return c
	})
}

//returns the length of s
func (s ConcurrentSeq) Len() int {
	len := 0
	Do(s, func(el El){
		len++
	})
	return len
}

//returns a new ConcurrentSeq that appends this one and s2
func (s ConcurrentSeq) Append(s2 Seq) Seq {return CAppend(s, s2)}

//returns a new ConcurrentSeq that appends s2 and this one
func (s ConcurrentSeq) Prepend(s2 Seq) Seq {return CAppend(s2, s)}

//returns a new ConcurrentSeq consisting of the elements of s for which filter returns true
func (s ConcurrentSeq) Filter(f func(e El)bool) Seq {return CFilter(s, f)}

//returns a new ConcurrentSeq consisting of the results of appying f to the elements of s
func (s ConcurrentSeq) Map(f func(i El)El) Seq {return CMap(s, f)}

//returns a new ConcurrentSeq consisting of the concatenation of the sequences f returns when applied to all of the elements of s
func (s ConcurrentSeq) FlatMap(f func(i El) Seq) Seq {return CFlatMap(s, f)}

//returns a new SequentialSeq constructed by recursively converting nested
//ConcurrentSeqs to SequentialSeqs.  Does not descend into nested sequential sequences
func (s ConcurrentSeq) ToSequentialSeq() *SequentialSeq {
	return SMap(s, func(el El)El{
		switch seq := el.(type) {case ConcurrentSeq: return seq.ToSequentialSeq()}
		return el
	})
}


// a sequential sequence
type SequentialSeq []interface{}

//returns a new SequentialSeq consisting of els
func From(els... interface{}) *SequentialSeq {return (*SequentialSeq)(&els)}

//returns a new SequentialSeq consisting of the numbers from 0 to limit, in succession
func AUpto(limit int) *SequentialSeq {
	a := make([]interface{}, limit)
	for i := 0; i < limit; i++ {
		a[i] = i
	}
	return (*SequentialSeq)(&a)
}

//returns the first item in a sequence for which f returns true or nil if none is found
func (s *SequentialSeq) Find(f func(el El)bool) El {
	for i := 0; i < len(*s); i++ {
		if f((*s)[i]) {return (*s)[i]}
	}
	return nil
}

//returns a new SequentialSeq consisting of all of the elements of s except for the first one
func (s *SequentialSeq) Rest() Seq {
	s2 := (*s)[1:]
	return (*SequentialSeq)(&s2)
}

//returns the length of s
func (s *SequentialSeq) Len() int {return len(*s)}

//returns a new SequentialSeq that appends this one and s2
func (s *SequentialSeq) Append(s2 Seq) Seq {return SAppend(s, s2)}

//returns a new SequentialSeq that appends s2 and this one
func (s *SequentialSeq) Prepend(s2 Seq) Seq {return SAppend(s2, s)}

//returns a new SequentialSeq consisting of the elements of s for which filter returns true
func (s *SequentialSeq) Filter(f func(e El)bool) Seq {return SFilter(s, f)}

//returns a new SequentialSeq consisting of the results of appying f to the elements of s
func (s *SequentialSeq) Map(f func(i El)El) Seq {return SMap(s, f)}

//returns a new SequentialSeq consisting of the concatenation of the sequences f returns when applied to all of the elements of s
func (s *SequentialSeq) FlatMap(f func(i El) Seq) Seq {return SFlatMap(s, f)}
