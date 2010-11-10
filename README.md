SEQ
===

The seq package provides functional sequences with the Seq interface.  Seq provides two concrete implementations: SequentialSeq and ConcurrentSeq.  SequentialSeq is like vector, and ConcurrentSeq is like a vector that calculates its elements in the background, which can be used for lazy sequences or for some types of parallel computation.

If you want to enumerate the elements of a Sequence, you can use one of the Sequence methods or you can just call it in the range clause of a for-loop, like this:

	Do(seq, func(el El){
		...
	})

The functionality is mainly accessible through functions, rather than methods, although there is partial method support.  The methods are mainly there to support the functions.  Please see the documentation for more information.
