---
layout: post
title:  "Third times the charm, more Haskell"
date:   2017-02-05 2:53:00:00 +1100
categories: haskell
---

Today I'm going to work through chapter 3 of [Exercises for Programmers: 57 Challenges to Develop Your Coding Skills](http://amzn.to/1T8eFEw) and have a crack at some numeric problems.

# Area of a Rectangular Room

> Create a program that calculates the area of a room. Prompt the user for the length and width of the room in feet. Then display the area in both square feet and square meters.
>
> __Constraints:__ Keep the calculations separate from the output. Use a constant to hold the conversion factor.

{% highlight hs %}
import System.IO

main = do

    let feet_meters_conversion_factor = 0.09290304

    putStr ("What is the length of the room in feet? ")
    hFlush stdout
    lengthStr <- getLine
    let length = read lengthStr :: Float
    
    putStr ("What is the width of the room in feet? ")
    hFlush stdout
    widthStr <- getLine
    let width = read widthStr :: Float
    
    putStrLn ("\nYou entered dimensions of " ++ lengthStr ++ " feet by " ++ widthStr ++ " feet.")
    putStrLn ("The area is:")
    
    let areaFeet = length * width
    let areaMeters = areaFeet * feet_meters_conversion_factor

    putStrLn (show areaFeet ++ " square feet")
    putStrLn (show areaMeters ++ " square meters")
{% endhighlight %}

That was simple, but I learned that I cannot name a constant using `UPPER_CASE_UNDERSCORE_NOTATION` like I normally would because variables must start with a lower case letter. Only data type names can beging with an upper case letter.

> Revise the program to ensure that inputs are entered as numeric values. Don't allow the user to proceed if the value entered is not numeric.


