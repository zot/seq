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
//import "reflect"

func add(i int, s Seq) Seq {
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
	d4 := add(1, AUpto(4))
	d6 := add(1, AUpto(6))
	d8 := add(1, AUpto(8))
	d10 := add(1, AUpto(10))
	names := map[interface{}]string{d4:"d4", d6:"d6", d8:"d8", d10:"d10"}
	dice := From(d4, d6, d8, d10)
	rank := map[Seq]int{d4:0, d6:1, d8:2, d10:3}
	sets := map[string]int{}
	//attempts is [[label, [score, ...]]...]
	attempts := Map(Filter(Product(From(dice, dice, dice)), func(d El)bool{
		oldRank := -1
		result := true
		Do(d.(Seq), func(set El){
			newRank := rank[set.(Seq)]
			result = result && newRank >= oldRank
			oldRank = newRank
		})
		return result
	}), func(el El)El{
		buf := bytes.NewBuffer(make([]byte, 0, 10))
		io.WriteString(buf, "<")
		Pretty(el.(Seq), names, buf)
		io.WriteString(buf, ">")
		return From(buf.String(), Map(Product(el.(Seq)), func(el El)El{
			return Fold(el.(Seq), 0, func(acc, el El)El{return max(acc.(int), el.(int))})
		}))
	
	})
	println("#sets:", len(sets))
	fmt.Println("#Attempts:", Len(attempts))
	println("results...")
	Do(attempts, func(el El){
		label, rolls := First2(el.(Seq))
		fmt.Printf("%s: %d\n", label, Len(rolls.(Seq)))
	})
	Do(CFlatMap(attempts, func(el El) Seq {
		label, sc := First2(el.(Seq))
		return CMap(attempts, func(del El) El {
			rolls, wins := 0, 0
			margins := map[int]int{}
			dlabel, dsc := First2(del.(Seq))
			Do(Product(From(sc,dsc)), func(rel El){
				rolls++
				attack, defense := First2(rel.(Seq))
				margin := attack.(int) - defense.(int)
				if margin > 0 {
					wins++
					margins[margin]++
				}
				
			})
			return From(label, dlabel, rolls, wins, margins)
		})
	}), func(el El){
		l, d, r, w, m := First5(el.(Seq))
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
