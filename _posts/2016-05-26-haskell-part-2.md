---
layout: post
title:  "The second Haskell session"
date:   2016-05-26 12:51:12 +1100
categories: haskell
---

It's time for some more Haskell. I'm going to work through chapter 2 of [Exercises for Programmers: 57 Challenges to Develop Your Coding Skills](http://amzn.to/1T8eFEw) and see what happens.

# Saying Hello

> Create a program that prompts for your name and prints a greeting using your name.
>
> __Constraints:__ Keep the input, string concatenation, and output separate.

{% highlight hs %}
import System.IO

main = do
    hSetBuffering stdout NoBuffering
    hSetBuffering stdin NoBuffering

    putStr ("What is your name? ")
    name <- getLine
    
    let greeting = "Hello, " ++ name ++ ", nice to meet you!"

    putStrLn (greeting)
{% endhighlight %}

Well, that was simpler than my first program, and there's nothing new to mention, so I'll move straight on to the first challenge.

> Write a new version of the program without using any variables

{% highlight hs %}
import System.IO

greet :: String -> String
greet name = "Hello, " ++ name ++ ", nice to meet you!"

main = do
    hSetBuffering stdout NoBuffering
    hSetBuffering stdin NoBuffering

    putStr "What is your name? "
    name <- getLine
    putStrLn (greet name)
{% endhighlight %}

I thought about this for a while, and I would like to solve it by writing functions and combining them. I can write a function `greet` to concatenate the name and greeting, but I'm not sure how to write the function that prints the question and returns the name that the user enters as a `String` instead of an `IO String`, so I'm still using one variable that binds to the value that `GetLine` produces. I'll come back to this once I learn some more about monads.

> Write a version of the program that displays different greetings for different people.

{% highlight hs %}
import System.IO

greet :: String -> String
greet "Bob" = "Hello, Bob, great to see you!"
greet "Alice" = "Hello, Alice, it's been too long!"
greet name = "Hello, " ++ name ++ ", nice to meet you!"

main = do
    hSetBuffering stdout NoBuffering
    hSetBuffering stdin NoBuffering

    putStr "What is your name? "
    name <- getLine
    putStrLn (greet name)
{% endhighlight %}

I extended the `greet` function to greet Bob and Alice differently by using pattern matching on the input parameter.

Let's move onto to the next part of the chapter.

# Counting the Number of Characters

> Create a program that prompts for an input string and displays output that shows the input string and the numer of characters the string contains.
>
> __Constraints:__ Be sure the output contains the original string. Use a single output statement to construct the output. Use a built-in function of the programming language to determine the length of the string.

{% highlight hs %}
import System.IO

main = do
    hSetBuffering stdout NoBuffering
    hSetBuffering stdin NoBuffering

    putStr "What is the input string? "
    input <- getLine
    putStrLn (input ++ " has " ++ show (length input) ++ " characters")
{% endhighlight %}

I use the built in `length` function to calculate the length of the input and because it's an `Integer` I use `show` to get its `String` representation.

> If the user enters nothing, state that the user must enter something into the program

{% highlight hs %}
import System.IO

prompt :: IO ()
prompt = do
    putStr "What is the input string? "
    hFlush stdout
    str <- getLine

    if length str > 0
        then putStrLn (str ++ " has " ++ show (length str) ++ " characters")
    else prompt

main = do
    prompt
{% endhighlight %}

Here I made a function called `prompt` that asks the user to input a string. If the length of the string is greater than 0 then I print the output, otherwise I recursively call `prompt` again to ask the user for input. I also saw how to use `hFlush stdout` when I need to flush the output, which is much nicer than the `hSetBuffering stdout NoBuffering` and `hSetBuffering stdin NoBuffering` I was doing before.

There is a third challenge to implement the program with a GUI, but I'm going to leave that for now until I learn some more Haskell!

# Printing Quotes

> Create a program that promts for a quote and an author. Display the quotation and author in the specified format.
>
> __Constraints:__ Use a single output statement to produce this output. Use string concatenation (not templates).

{% highlight hs %}
import System.IO

main = do
    putStr "What is the quote? "
    hFlush stdout
    quote <- getLine
    putStr "Who said it? "
    hFlush stdout
    author <- getLine
    
    putStrLn (author ++ " says, \"" ++ quote ++ "\"")
{% endhighlight %}

This is very similar to the other programs. I had to escape the double quotes with a `\`.

> Modify this program so that instead of prompting for quotes from the user, you create a structure that holds quotes  and their associated attributions and then display all of the quotes using the specified format.

{% highlight hs %}
import System.IO

quotes :: [(String, String)]
quotes =
    [("Obi Wan Kenobi", "These aren't the droids you're looking for."),
    ("Neil Armstrong", "Houston, Tranquility Base here. The Eagle has landed."),
    ("Arnold Schwarzenegger", "I'll be back!")]

printQuote :: (String, String) -> IO ()
printQuote (author, quote) = putStrLn (author ++ " says, \"" ++ quote ++ "\"")

main = do
    mapM_ printQuote quotes
{% endhighlight %}

This was an interesting modification. First I stored the quotes as a list of tuples with two `String` elements. Then I found the `mapM_` function that takes a data structure and a monadic action and applies the action to each element. So I wrote a little function called `printQuote` to take an (author, quote) tuple and print it with some formatting. Then in `main` I call `mapM_` with two arguments, my printing function and the list of quotes, and behold, the quotes are printed.

I feel like I've gone a little far without learning some more fundamentals at this stage, but the chapter is almost over, only one more program to write.

# Mad Lib

> Create a simple mad-lib program that prompts for a noun, a verb, an adverb, and an adjective and injects those into a story that you create.
>
> __Constraints:__ Use a single output statement for this program. If your language supports string interpolation or string substitution, use it to build up the output

{% highlight hs %}
import System.IO
import Text.Printf

main = do
    putStr "Enter a noun: "
    hFlush stdout
    noun <- getLine
    putStr "Enter a verb: "
    hFlush stdout
    verb <- getLine
    putStr "Enter an adjective: "
    hFlush stdout
    adjective <- getLine
    putStr "Enter an adverb: "
    hFlush stdout
    adverb <- getLine
    
    let madlib = "If you %s your %s %s %s I will eat my hat!"
    
    printf madlib verb adjective noun adverb
{% endhighlight %}

I found a `printf` function to do the string interpolation.

There's a challenge to add more inputs to the program to expand the story, but I'm going to stop here and move on to the next chapter to work on some numeric problems and learn some more tricks before I try to write a larger program.

