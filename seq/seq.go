// Copyright 2010 Bill Burdick. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
package seq

import "fmt"
import "io"
import "os"
import "container/vector"

type Element interface{}
type SeqChan chan Element
type Sequence func()SeqChan

func IsSeq(s interface{}) bool {
	_, test := s.(Sequence)
	return test
}

func Seq(f func(c SeqChan)) Sequence {
	return func() SeqChan {
		c := make(SeqChan)
		go func() {
			defer close(c)
			f(c)
		}()
		return c
	}
}

func From(el... interface{}) Sequence {
	return Seq(func(c SeqChan) {Output(c, el...)})
}

func Output(c SeqChan, el... interface{}) {
	for _, v := range el {
		c <- v
	}
}

func Bleed(c SeqChan) {
	for !closed(c) {
		<- c
	}
}

func (s Sequence) First() Element {
	c := s()
	defer Bleed(c)
	return <- c
}

func (s Sequence) Rest() Sequence {
	return func()SeqChan{
		c := s()
		<- c
		return c
	}
}

func (s Sequence) AddFirst(els... interface{}) Sequence {
	return Seq(func(c SeqChan){
		for el := range els {
			c <- el
		}
		s.Output(c)
	})
}

func (s Sequence) AddLast(els... interface{}) Sequence {
	return Seq(func(c SeqChan){
		s.Output(c)
		for el := range els {
			c <- el
		}
	})
}

func (s Sequence) IsEmpty() bool {
	c := s()
	<- c
	result := closed(c)
	if !result {defer Bleed(c)}
	return result
}

func (s Sequence) Append(s2 Sequence) Sequence {
	return Seq(func(c SeqChan){
		s.Output(c)
		s2.Output(c)
	})
}

func (s Sequence) Len() int {
	len := 0
	c := s()
	for !closed(c) {
		<- c
		len++
	}
	return len - 1
}

func (s Sequence) Output(c chan Element) {
	for el := range s() {
		c <- el
	}
}

func Upto(limit int) Sequence {
	return Seq(func(c SeqChan) {
		for i := 0; i < limit; i++ {
			c <- i
		}
	})
}

func (s Sequence) Filter(filter func(e Element)bool) Sequence {
	return Seq(func(c SeqChan){
		for el := range s() {
			if filter(el) {
				c <- el
			}
		}
	})
}

func (s Sequence) Do(f func(el Element)) {
	for v := range s() {
		f(v)
	}
}

func (s Sequence) Map(f func(i Element)Element) Sequence {
	return Seq(func(c SeqChan) {
		for v := range s() {
			c <- f(v)
		}
	})
}

func (s Sequence) FlatMap(f func(i Element) Sequence) Sequence {
	return Seq(func(c SeqChan) {
		for v := range s() {
			for sub := range f(v)() {
				c <- sub
			}
		}
	})
}

func (s Sequence) Fold(init Element, f func(acc, el Element)Element) Element {
	for el := range s() {
		init = f(init, el)
	}
	return init
}

//maybe convert this to use an accumulator instead of append?
func (s Sequence) Combinations(number int) Sequence {
	if number == 0 || s.IsEmpty() {return From(From())}
	return s.Rest().Combinations(number - 1).Map(func(el Element)Element{
		return el.(Sequence).AddFirst(s.First())
	}).Append(s.Rest().Combinations(number))
}

//returns the product of the Sequences contained in sequences
func (sequences Sequence) Product() Sequence {
	return sequences.Fold(From(From()), func(acc, el Element)Element{
		return el.(Sequence).peelOnto(acc.(Sequence))
	}).(Sequence)
}

func (s Sequence) peelOnto(seq Sequence) Sequence {
	return seq.FlatMap(func(old Element)Sequence{
		return s.Map(func(i Element) Element {
			return old.(Sequence).Append(From(i))
		})
	})
}

func (s Sequence) Reify() Sequence {
	vec := vector.Vector(make([]interface{}, 0, 128))
	for v := range s() {
		sv, is := v.(Sequence)
		if is {
			vec.Push(sv.Reify())
		} else {
			vec.Push(v)
		}
	}
	return From([]interface{}(vec)...)
}

func (s Sequence) Prettyln(names map[Element]string, writer... io.Writer) {
	if len(writer) == 0 {
		writer = []io.Writer{os.Stdout}
	}
	s.Pretty(names, writer...)
	fmt.Fprintln(writer[0])
}
func (s Sequence) Pretty(names map[Element]string, writer... io.Writer) {
	if len(writer) == 0 {
		writer = []io.Writer{os.Stdout}
	}
	s.prettyLevel(0, names, writer[0])
}

//This is pretty ugly :)
func (s Sequence) prettyLevel(level int, names map[Element]string, w io.Writer) {
	name, hasName := names[s]
	if hasName {
		fmt.Fprint(w, name)
	} else {
		c := s()
		fmt.Fprintf(w, "%*s%s", level, "", "[")
		first := true
		innerSeq := false
		named := false
		for v := range c {
			_, named = names[v]
			_,innerSeq = v.(Sequence)
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
				v.(Sequence).prettyLevel(level + 4, names, w)
			} else {
				fmt.Fprintf(w, "%v", v)
			}
		}
		if innerSeq {
			if !named {
				fmt.Fprintf(w, "\n%*s", level, "")
			}
		}
		fmt.Fprintf(w, "]")
	}
}
