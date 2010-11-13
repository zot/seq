// Copyright 2010 Bill Burdick. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
package seq

import "fmt"
import "io"
import "os"
import "reflect"

// convenience alias for sequence elements
type El interface{}
// main type
type Sequence struct {
	S Seq
}
// interface for basic sequence support
type Seq interface {
	Find(f func(i El)bool) El
	Rest() Seq
	Len() int
	IsConcurrent() bool
}

//convert a sequence to a concurrent sequence (if necessary)
func (s Sequence) Concurrent() Sequence {
	switch seq := s.S.(type) {case ConcurrentSeq: return s}
	return Gen(func(c SeqChan){s.Output(c)})
}

//convert a sequence to a sequential sequence (if necessary)
func (s Sequence) Sequential() Sequence {
	switch seq := s.S.(type) {case *SequentialSeq: return s}
	return s.SMap(func(el El)El{return el})
}

//returns a new array of the first N items
func (s Sequence) FirstN(n int) []interface{} {
	r := make([]interface{}, n)
	x := 0
	s.Find(func(el El)bool{
		r[x] = el
		x++
		return x == n
	})
	return r
}

//convenience function with multiple return values
func (s Sequence) First2() (a, b interface{}) {
	r := s.FirstN(2)
	return r[0], r[1]
}

//convenience function with multiple return values
func (s Sequence) First3() (a, b, c interface{}) {
	r := s.FirstN(3)
	return r[0], r[1], r[2]
}

//convenience function with multiple return values
func (s Sequence) First4() (a, b, c, d interface{}) {
	r := s.FirstN(4)
	return r[0], r[1], r[2], r[3]
}

//convenience function with multiple return values
func (s Sequence) First5() (a, b, c, d, e interface{}) {
	r := s.FirstN(5)
	return r[0], r[1], r[2], r[3], r[4]
}

//convenience function with multiple return values
func (s Sequence) First6() (a, b, c, d, e, f interface{}) {
	r := s.FirstN(6)
	return r[0], r[1], r[2], r[3], r[4], r[5]
}

//returns whether s can be interpreted as a sequence
func IsSeq(s interface{}) bool {
	_, test := s.(Seq)
	return test
}

//returns the first item in a sequence
func (s Sequence) First() interface{} {
	var result interface{}
	s.S.Find(func(el El)bool{
		result = el
		return true
	})
	return result
}

//returns whether a sequence is empty
func (s Sequence) IsEmpty() bool {
	empty := true
	s.S.Find(func(el El)bool{
		empty = false
		return true
	})
	return empty
}

//returns the first item in a sequence for which f returns true or nil if none is found
func (s Sequence) Find(f func(el El) bool) El {return s.S.Find(f)}

//applies f to each item in the sequence until f returns false
func (s Sequence) While(f func(el El) bool) {s.S.Find(func(el El)bool{return !f(el)})}

//applies f to each item in the sequence
func (s Sequence) Do(f func(el El)) {
	s.S.Find(func(el El)bool{
		f(el)
		return false
	})
}

//applies f concurrently to each element of s, in no particular order; sizePowerOpt will default to {6} and CMap will allow up to 1 << sizePowerOpt[0] outstanding concurrent instances of f at any time
func (s Sequence) CDo(f func(el El), sizePowerOpt... uint) {
	c := s.CMap(func(el El)El{f(el); return nil}, sizePowerOpt...).S.(ConcurrentSeq)()
	for <- c; !closed(c); <- c {}
}

//returns the length of s
func (s Sequence) Len() int {return s.S.Len()}

//sends each item of s to c
func (s Sequence) Output(c SeqChan) {s.Do(func(el El){c <- el})}

//returns a new sequence of the same type as s consisting of all of the elements of s except for the first one
func (s Sequence) Rest() Sequence {return Sequence{s.S.Rest()}}

//returns a new sequence of the same type as s1 that appends this s1 and s2
func (s1 Sequence) Append(s2 Sequence) Sequence {
	if s1.S.IsConcurrent() {return s1.CAppend(s2)}
	return s1.SAppend(s2)
}

//returns a new sequence of the same type as s1 that appends this s1 and s2
func (s1 Sequence) Prepend(s2 Sequence) Sequence {
	if s1.S.IsConcurrent() {return s2.CAppend(s1)}
	return s2.SAppend(s1)
}

func (s Sequence) ToSlice() []interface{} {
	return *(*[]interface{})(s.Sequential().S.(*SequentialSeq))
}

//returns a new SequentialSeq which consists of appending s and s2
func (s Sequence) SAppend(s2 Sequence) Sequence {
	slice := s.ToSlice()
	slice = append(slice, s2.ToSlice()...)
	return Sequence{(*SequentialSeq)(&slice)}
}

//returns a new ConcurrentSeq which consists of appending s and s2
func (s Sequence) CAppend(s2 Sequence) Sequence {
	return Gen(func(c SeqChan){
		s.Output(c)
		s2.Output(c)
	})
}

//if s is a SequentialSeq, return its length, otherwise return d
func (s Sequence) quickLen(d int) int {
	switch seq := s.S.(type) {case *SequentialSeq: return s.Len()}
	return d
}

//returns a new sequence of the same type as s consisting of the elements of s for which filter returns true
func (s Sequence) Filter(filter func(e El)bool) Sequence {
	if s.S.IsConcurrent() {return s.CFilter(filter)}
	return s.SFilter(filter)
}

func ifFunc(condition func(e El)bool, op func(e El)) func(el El){return func(el El){if condition(el) {op(el)}}}

//returns a new SequentialSeq consisting of the elements of s for which filter returns true
func (s Sequence) SFilter(filter func(e El)bool) Sequence {
	//continue shrinking
	slice := make([]interface{}, 0, s.quickLen(8))
	s.Do(ifFunc(filter, func(el El){slice = append(slice, el)}))
	return Sequence{(*SequentialSeq)(&slice)}
}

//returns a new ConcurrentSeq consisting of the elements of s for which filter returns true; sizePowerOpt will default to {6} and CMap will allow up to 1 << sizePowerOpt[0] outstanding concurrent instances of f at any time
func (s Sequence) CFilter(filter func(e El)bool, sizePowerOpt... uint) Sequence {
	return Gen(func(c SeqChan){
		s.CDo(ifFunc(filter, func(el El){c <- el}), sizePowerOpt...)
	})
}

//returns a new sequence of the same type as s consisting of the results of appying f to the elements of s
func (s Sequence) Map(f func(el El) El) Sequence {
	if s.S.IsConcurrent() {return s.CMap(f)}
	return s.SMap(f)
}

//returns a new SequentialSeq consisting of the results of appying f to the elements of s
func (s Sequence) SMap(f func(i El)El) Sequence {
	slice := make([]interface{}, 0, s.quickLen(8))
	s.Do(func(el El){slice = append(slice, f(el))})
	return Sequence{(*SequentialSeq)(&slice)}
}

type reply struct {
	index int;
	result El
}

type swEntry struct {
	value El
	present bool
}

// like a slice of a sparse vector where capacity is always a power of 2
// it uses a ring buffer so RemoveFirst is efficient
type SlidingWindow struct {
	start, base, count, mask int
	values []swEntry
}
//creates a new SlidingWindow with capacity size
func NewSlidingWindow(sz uint) *SlidingWindow {return &SlidingWindow{0, 0, 0, (1 << sz) - 1, make([]swEntry, 1 << sz)}}
//returns the current maximum available index
func (r *SlidingWindow) Max() int {return r.base + len(r.values) - 1}
//returns the size of the window
func (r *SlidingWindow) Capacity() int {return len(r.values)}
//returns the number of items in the window
func (r *SlidingWindow) Count() int {return r.count}
func (r *SlidingWindow) normalize(index int) int {return (index + len(r.values)) & r.mask}
//returns whether the window is empty
func (r *SlidingWindow) IsEmpty() bool {return r.count == 0}
//returns whether the window has any available space
func (r *SlidingWindow) IsFull() bool {return r.count == len(r.values)}
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
func (s Sequence) CMap(f func(el El) El, sizePowerOpt... uint) Sequence {
// spawn a goroutine that does the following for each value, with up to size pending at a time:
//   spawn a goroutine to apply f to the value and send the result back in a channel
// send the results in order to the ouput channel as they are completed
	sizePower := uint(6)
	if len(sizePowerOpt) > 0 {sizePower = sizePowerOpt[0]}
	size := 1 << sizePower
	return Gen(func(output SeqChan){
		//punt and convert sequence to concurrent
		//maybe someday we'll handle SequentialSequences separately
		input := s.Concurrent().S.(ConcurrentSeq)()
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
func (s Sequence) FlatMap(f func(el El) Sequence) Sequence {
	if s.S.IsConcurrent() {return s.CFlatMap(f)}
	return s.SFlatMap(f)
}

//returns a new SequentialSeq consisting of the concatenation of the sequences f returns when applied to all of the elements of s
func (s Sequence) SFlatMap(f func(i El) Sequence) Sequence {
	slice := make([]interface{}, 0, s.quickLen(8))
	s.Do(func(e El){f(e).Do(func(sub El){slice = append(slice, sub)})})
	return Sequence{(*SequentialSeq)(&slice)}
}

//returns a new ConcurrentSeq consisting of the concatenation of the sequences f returns when applied to all of the elements of s; sizePowerOpt will default to {6} and CMap will allow up to 1 << sizePowerOpt[0] outstanding concurrent instances of f at any time
func (s Sequence) CFlatMap(f func(i El) Sequence, sizePowerOpt... uint) Sequence {
	return Gen(func(c SeqChan){
		s.CMap(func(e El)El{return f(e)}, sizePowerOpt...).Do(func(sub El){
			sub.(Sequence).Output(c)
		})
	})
}

//returns the result of applying f to its previous value and each element of s in succession, starting with init as the initial "previous value" for f
func (s Sequence) Fold(init interface{}, f func(acc, el El)El) interface{} {
	s.Do(func(el El){init = f(init, el)})
	return init
}

//returns a new sequence of the same type as s consisting of all possible combinations of the elements of s of size number or smaller
func (s Sequence) Combinations(number int) Sequence {
	if number == 0 || s.IsEmpty() {return From(From())}
	return s.Rest().Combinations(number).Prepend(s.Rest().Combinations(number - 1).Map(func(el El)El{
		return el.(Sequence).Prepend(From(s.First()))
	}))
}

//returns the product of the elements of sequences, where each element is a sequence
func (sequences Sequence) Product() Sequence {
	return sequences.Fold(From(From()), func(result, each El)El{
		return result.(Sequence).FlatMap(func(seq El)Sequence{
			return each.(Sequence).Map(func(i El) El {
				return seq.(Sequence).Append(From(i))
			})
		})
	}).(Sequence)
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

func hashable(v interface{}) bool {
	k := reflect.Typeof(v).Kind()
	return k < reflect.Array || k == reflect.Ptr || k == reflect.UnsafePointer
}

func getName(names map[interface{}]string, v interface{}) (string, bool) {
	_, seq := v.(Sequence)
	if seq {
		v = v.(Sequence).S
	}
	if hashable(v) {
		kk, vv := names[v]
		return kk, vv
	}
	return "", false
}

func hasName(names map[interface{}]string, v interface{}) bool {
	_, has := getName(names, v)
	return has
}

//This pretty is ugly :)
func prettyLevel(s interface{}, level int, names map[interface{}]string, w io.Writer) {
	name, has := getName(names, s)
	if has {
		fmt.Fprint(w, name)
	} else switch arg := s.(type) {
	case Sequence: prettyLevel(arg.S, level, names, w)
	case Seq:
		fmt.Fprintf(w, "%*s%s", level, "", "[")
		first := true
		innerSeq := false
		named := false
		Sequence{arg}.Do(func(v El) {
			named = hasName(names, v)
			_,innerSeq = v.(Sequence)
			if first {
				first = false
				if !named && innerSeq {fmt.Fprintln(w)}
			} else if !named && innerSeq {
				fmt.Fprintln(w, ",")
			} else {
				fmt.Fprint(w, ", ")
			}
			if innerSeq {
				prettyLevel(v.(Sequence), level + 4, names, w)
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
func Gen(f func(c SeqChan)) Sequence {
	return Sequence{ConcurrentSeq(func() SeqChan {
		c := make(SeqChan)
		go func() {
			defer close(c)
			f(c)
		}()
		return c
	})}
}

//returns a new ConcurrentSeq consisting of the numbers from 0 to limit, in succession
func CUpto(limit int) Sequence {
	return Sequence(Gen(func(c SeqChan) {
		for i := 0; i < limit; i++ {
			c <- i
		}
	}))
}

//ConcurrentSeqs are concurrent; return true
func (s ConcurrentSeq) IsConcurrent() bool {return true}

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
	s.Find(func(el El)bool{
		len++;
		return false
	})
	return len
}

//returns a new SequentialSeq constructed by recursively converting nested
//ConcurrentSeqs to SequentialSeqs.  Does not descend into nested sequential sequences
func (s ConcurrentSeq) ToSequentialSeq() Sequence {
	return Sequence{s}.SMap(func(el El)El{
		switch seq := el.(type) {case ConcurrentSeq: return seq.ToSequentialSeq()}
		return el
	})
}


// a sequential sequence
type SequentialSeq []interface{}

//returns a new SequentialSeq consisting of els
func From(els... interface{}) Sequence {return Sequence{(*SequentialSeq)(&els)}}

//returns a new SequentialSeq consisting of the numbers from 0 to limit, in succession
func SUpto(limit int) Sequence {
	a := make([]interface{}, limit)
	for i := 0; i < limit; i++ {
		a[i] = i
	}
	return Sequence{(*SequentialSeq)(&a)}
}

//SequentialSeqs are not concurrent; return false
func (s *SequentialSeq) IsConcurrent() bool {return false}

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
