---
layout: post
title:  "Learn you a Haskell"
date:   2016-05-10 15:26:12 +1100
categories: haskell
---

I've been reading about functional programming lately, and I want to learn a new language to try it out properly. I picked Haskell for a few reasons. First, it's touted as a [pure](https://en.wikipedia.org/wiki/Purely_functional) functional language, and that appeals to me because I want to learn functional programming in particular, and not have the concepts blurred by a hybrid language. Second, there's a lot of good online resources that I can learn from. Third, it has a cool name.

There's a couple of online tutorials I've found that seem to be well regarded. I'm working through them in parallel. [Learn You A Haskell](http://learnyouahaskell.com/) is a little more wordy (and funny!), [Real World Haskell](http://book.realworldhaskell.org/) has more practical examples. Between the two of them I'm beginning to understand things. I also bought a new book called [Exercises for Programmers: 57 Challenges to Develop Your Coding Skills](https://www.amazon.com/Exercises-Programmers-Challenges-Develop-Coding/dp/1680501224) to help me with some ideas for small programs to write and test my knowlege.

So without further ado, here is the first question in the book, and my first program in Haskell.

> Create a simple tip calculator. The program should prompt for a bill amount and a tip rate. The program must compute the tip and display both the tip and the total amount of the bill.

{% highlight hs %}
import System.IO

main = do
    hSetBuffering stdout NoBuffering
    hSetBuffering stdin NoBuffering

    putStr "What is the bill amount? "
    billAmountInput <- getLine
    putStr "What is the tip rate? "
    tipRateInput <- getLine

    let billAmount = read billAmountInput :: Float
    let tipRate = read tipRateInput :: Float

    let tip = billAmount * tipRate
    let total = billAmount + tip;

    putStrLn ("The tip is $" ++ show tip)
    putStrLn ("The total is $" ++ show total)
{% endhighlight %}

To run this program in the `ghci` interpreter:

* save the code to a file called `tip.hs`
* open `ghci` at the location of the file
* type `:l tip.hs` at the prompt to load the file
* type `main` to run the program

Alternatively you can compile it:

* save the code to a file called `tip.hs`
* type `ghc --make tip.hs` to compile the program
* type `tip.exe` (or `./tip` depending on your environment) to run the program

This exercise stretched me further than I had read in either of the tutorials, so I had to scrap around a bit and learn how to do basic IO (monads). So how does it work? Well ...

I import the `System.IO` library. I didn't realise this at first, but the input and output were buffered by default and all of the calls to `PutStr` were being printed at once which was undesirable, so I'm going to disable buffering for `stdout` and `stdin` and for that I need `System.IO`.

I define a function called `main` that executes a sequence of actions in a `do` block. I disable buffering for `stdout` and `stdin` as previously mentioned. The program prompts for the bill amount and the tip rate and stores the input. I use `read` to convert the input from a `String` to a `Float`. Then I calculate the tip and the total and write them out to the console using `++` to concatenate and `show` to convert the `Float` value to a `String`.

It's a bit rough, I admit it. There are no constraints or error checking on the input, `read` will throw an exception if the input `String` can't be parsed to a `Float`. Strange things might happen if the `Float` calculation overflows. I don't print the values out with 2 decimal places, and there's nothing particularly functional about any of it, but that right there is my first Haskell program, I learned a bunch of new things and I'm a little chuffed!