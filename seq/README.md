SEQ
===

Seq provides Sequence, a lazy, concurrent container.  A Sequence is a function which returns a new channel and starts a new goroutine which applies a function to the channel and then closes it.  If you want to enumerate the elements of a Sequence, you can use one of the Sequence methods or you can just call it in the range clause of a for-loop, like this:

	for el := range seq() {
		...
	}

Sequence supports the following operations:

* Seq(f func(c chan Element)) Sequence
		returns a new Sequence, a function which returns a new channel and runs a goroutine that applies f to the channel and then closes the channel
* From(el... interface{}) Sequence

		returns a new Sequence of el
* Upto(limit int) Sequence

		returns a new Sequence of the numbers from 0 to limit
* (s Sequence) First() Sequence

		returns the first element of s
* (s Sequence) Rest() Sequence

		returns a new Sequence of the elements of s after the first
* IsSeq(el interface{}) bool

		returns whether an object is a Sequence
* (s Sequence) IsEmpty() bool

		returns whether there are any elements in s
* (s Sequence) Len() int

		returns the length of s
* (s Sequence) AddFirst(els... Element) Sequence

		returns a new Sequence of els, followed by the elements of s
* (s Sequence) AddLast(els... Element) Sequence

		returns a new Sequence of the elements of s, followed by els
* (s Sequence) Append(s2 Sequence) Sequence

		returns a new Sequence of the elements of s, followed by the elements of s2
* (s Sequence) Map(f func(el Element) Element) Sequence

		returns a new Sequence of the results of f applied to each element of s
* (s Sequence) FlatMap(f func(el Element) Sequence) Sequence

		returns a new Sequence of the concatenation of the results of f applied to each element of s
* (s Sequence) Filter(f func(el Element) bool) Sequence

		returns a new Sequence of the elements of s for which f returned true
* (s Sequence) Fold(init Element, f func(acc, el Element) Element) Element

		apply f to each element of s, along with the return value of the previous evaluation, returns the final value
* (s Sequence) Do(f func(el Element))

		apply f to each element of s
* (s Sequence) Combinations(number int) Sequence

		returns a new Sequence containing all combinations of 0 - number elements of s
* (s Sequence) Product(sequences Sequence) Sequence

		returns the product of the Sequences contained in sequences
* (s Sequence) Reify() Sequence

		returns a new Sequence containing the elements of s, computed by recursively walking s and all of the sequences it contains.  This is useful to cache s if it is based on expensive computation
* (s Sequence) Pretty(names map[Element]string, writer io.Writer)

		print s to writer (defaults to Stdout) in a "pretty" way.  Prints Elements which are contained in names are printed as the name
* (s Sequence) Prettyln(names map[Element]string, writer io.Writer)

		calls Pretty and prints a newline afterwards
* Output(c SeqChan, el... interface{})

		output all of the elements of s to the channel
* (s Sequence) Output(c SeqChan)

		output all of the elements of s to the channel
* Bleed(c SeqChan)

		read the remaining elements of c
