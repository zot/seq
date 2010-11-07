// Copyright 2010 Bill Burdick. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// dice is a program that uses seq to calculate some probabilities for a game:
// each player rolls 3 dice, each of which can have 4, 6, 8, or 10 sides
// then, they compare the maximum numbers.  The one with the highest number wins
// this program calculates probabilities of winning by a certain margin (which can be from 1 - 9)
package main

import "fmt"
import "io"
import "bytes"
import "container/vector"
import "sort"
import . "github.com/zot/bills-tools/seq"

func score(s Element) int {
	return s.(Sequence).Fold(0, func(acc, i Element)Element{
		if acc.(int) > i.(int) {
			return acc
		}
		return i
	}).(int)
}

func add(i int, s Sequence) Sequence {
	return s.Map(func(el Element)Element {
		return i + el.(int)
	})
}

func hist(scores Sequence) int {
	scorelen := scores.Len()
	fmt.Printf("%10d scores\n", scorelen)
	hist := map[int]int{}
	for i,v := range make([]int, 10) {
		hist[i + 1] = v
	}
	for i := range scores() {
		hist[i.(int)]++
	}
	for i := 1; i <= 10; i++ {
		percent := float(hist[i])*100/float(scorelen)
		fmt.Printf("%10d % 5.1f (%4d) ", i, percent, hist[i])
		for dot := 0; float(dot) < float(percent); dot++ {
			fmt.Printf(".")
		}
		fmt.Println()
	}
	return scorelen
}

func pair(seq Sequence) (Element, Element) {
	c := seq()
	defer close(c)
	a := <- c
	return a, <-c
}

func stamp(s Sequence, stamp Sequence) Sequence {
	return s.Map(func(el Element)Element{
		return From(stamp, el)
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	d4 := add(1, Upto(4))
	d6 := add(1, Upto(6))
	d8 := add(1, Upto(8))
	d10 := add(1, Upto(10))
	names := map[Element]string{d4:"d4", d6:"d6", d8:"d8", d10:"d10"}
	dice := From(d4, d6, d8, d10)
	rank := map[Sequence]int{d4:0, d6:1, d8:2, d10:3}
	sets := map[string]int{}
	//attempts is [[label, [score, ...]]...]
	attempts := From(dice, dice, dice).Product().Filter(func(d Element)bool{
		oldRank := -1
		result := true
		for set := range d.(Sequence)() {
			newRank := rank[set.(Sequence)]
			result = result && newRank >= oldRank
			oldRank = newRank
		}
		return result
	}).Map(func(el Element)Element{
		buf := bytes.NewBuffer(make([]byte, 0, 10))
		io.WriteString(buf, "<")
		el.(Sequence).Pretty(names, buf)
		io.WriteString(buf, ">")
		return From(buf.String(), el.(Sequence).Product().Map(func(el Element)Element{
			return el.(Sequence).Fold(0, func(acc, el Element)Element{return max(acc.(int), el.(int))})
		}))
	
	}).Reify()
	println("#sets:", len(sets))
	println("#Attempts:", attempts.Len())
	println("results...")
	attempts.Do(func(el Element){
		label, rolls := pair(el.(Sequence))
		fmt.Printf("%s: %d\n", label, rolls.(Sequence).Len())
	})
//	attempts.Prettyln(names)
	//scores is [score, ...]
	scores := attempts.FlatMap(func(el Element)Sequence{
		_, sc := pair(el.(Sequence))
		return sc.(Sequence)
	}).Reify()
	numScores := scores.Len()
	println("#scores:", numScores)
//	scores.Prettyln(names)
	attempts.Do(func(el Element){
		label, sc := pair(el.(Sequence))
		attempts.Do(func(del Element){
			rolls := 0
			wins := 0
			margins := map[int]int{}
			dlabel, dsc := pair(del.(Sequence))
			From(sc,dsc).Product().Do(func(rel Element){
				rolls++
				attack, defense := pair(rel.(Sequence))
				margin := attack.(int) - defense.(int)
				if margin > 0 {
					wins++
					margins[margin]++
				}
			})
			fmt.Printf("%s vs %s rolls: %d wins: %d margins:", label, dlabel, rolls, wins)
			for i := 1; i <= 9; i++ {
				v := margins[i]
				if v > 0 {
					fmt.Printf(" %d %.2f", v, float(v)*100/float(wins))
				}
			}
			println()
			dumpMargin(wins, margins)
		})
	})
}

func round(value float) int {
	floor := int(value)
	if value - float(floor) > 0.5 {
		if value > 0 {
			return floor + 1
		}
		return floor - 1
	}
	return floor
}

func dumpMargin(totMargin int, margins map[int]int) {
	for k := 1; k <= 9; k++ {
		v := margins[k]
		if v > 0 {
			percent := float(v)*100/float(totMargin)
			fmt.Printf("%d: %10d (%6.2f) ", k, v, percent)
			for i := 0; i < int(round(percent)); i++ {
				print(".")
			}
			println()
		}
	}
}
func dumpResults(totMargin int, margins map[string]map[int]int) {
	vec := vector.StringVector(make([]string, 0, 32))
	for k := range margins {
		vec.Push(k)
	}
	sort.StringArray(vec).Sort()
	for _, dice := range vec {
		println("Margins for", dice)
		dumpMargin(totMargin, margins[dice])
	}
}
