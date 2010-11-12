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
import . "github.com/zot/seq"
//import "reflect"

func add(i int, s Sequence) Sequence {
	return s.Map(func(el El)El {
		return i + el.(int)
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	d4 := add(1, SUpto(4))
	d6 := add(1, SUpto(6))
	d8 := add(1, SUpto(8))
	d10 := add(1, SUpto(10))
	names := map[interface{}]string{d4.S:"d4", d6.S:"d6", d8.S:"d8", d10.S:"d10"}
	dice := From(d4, d6, d8, d10)
	rank := map[Seq]int{d4.S:0, d6.S:1, d8.S:2, d10.S:3}
	sets := map[string]int{}
	//attempts is [[label, [score, ...]]...]
	attempts := From(dice, dice, dice).Product().Filter(func(d El)bool{
		oldRank := -1
		result := true
		// change this to a fold!
		d.(Sequence).Do(func(set El){
			newRank := rank[set.(Sequence).S]
			result = result && newRank >= oldRank
			oldRank = newRank
		})
		return result
	}).Map(func(el El)El{
		buf := bytes.NewBuffer(make([]byte, 0, 10))
		io.WriteString(buf, "<")
		Pretty(el.(Sequence), names, buf)
		io.WriteString(buf, ">")
		return From(buf.String(), el.(Sequence).Product().Map(func(el El)El{
			return el.(Sequence).Fold(0, func(acc, el El)El{return max(acc.(int), el.(int))})
		}))
	})
	println("#sets:", len(sets))
	fmt.Println("#Attempts:", attempts.Len())
	println("results...")
	attempts.Do(func(el El){
		label, rolls := el.(Sequence).First2()
		fmt.Printf("%s: %d\n", label, rolls.(Sequence).Len())
	})
	attempts.CFlatMap(func(el El) Sequence {
		label, sc := el.(Sequence).First2()
		return attempts.CMap(func(del El) El {
			rolls, wins := 0, 0
			margins := map[int]int{}
			dlabel, dsc := del.(Sequence).First2()
			From(sc,dsc).Product().Do(func(rel El){
				rolls++
				attack, defense := rel.(Sequence).First2()
				margin := attack.(int) - defense.(int)
				if margin > 0 {
					wins++
					margins[margin]++
				}
				
			})
			return From(label, dlabel, rolls, wins, margins)
		})
	}).Do(func(el El){
		l, d, r, w, m := el.(Sequence).First5()
		printResult(l.(string), d.(string), r.(int), w.(int), m.(map[int]int))
	})
}

func printResult(label string, dlabel string, rolls int, wins int, margins map[int]int) {
	fmt.Printf("%s vs %s rolls: %d wins: %d margins:", label, dlabel, rolls, wins)
	for i := 1; i <= 9; i++ {
		v := margins[i]
		if v > 0 {
			fmt.Printf(" %d %.2f", v, float(v)*100/float(wins))
		}
	}
	println()
	dumpMargin(wins, margins)
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
