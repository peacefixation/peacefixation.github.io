---
layout: post
title:  "Add an offset to a list of numbers with Vim"
date:   2017-08-30 21:00:00 +1000
categories: development
---


Given a file containing a list of numbers on consecutive lines, I want to add an offset of 10 to every number.

1<br />
2<br />
3<br />
4<br />
5<br />
6<br />
7<br />
8<br />
9<br />

First a quick Vim refresher. Vim has modes. Normal mode is for navigation and text manipulation, everything you type is interpreted as a command. You can use commands like `gg` to go the start of the file, `G` to go to the end of the file, `$` to go to the end of the line, etc. You can return to Normal mode from another mode by pressing `ESC`. Command-line mode is for Vim commands. You enter Command-line mode by pressing `:`, then you enter your command. For example `:set nu` turns line numbering on. `:help  Command-line-mode` opens the help file for Command-line-mode. With that out of the way, we are going to record a macro and then execute it by running a normal mode command from Command-line mode with the `normal` command. Vim is complicated. :)

Create a file with the numbers 1-9 on consequtive lines. Press `ESC` to make sure we're in normal mode. Now we will record a macro called `a`:

- press `qa` to start recording
- type `10` and press `Ctrl-A` to increment the first number by 10
- press `q` to stop recording

You can use `:u` to undo the last command to revert that change, alternatively type `10` and press `Ctrl-X` to decrement the number by 10.

Now we will apply the macro to every line in the file:

- enter `:%normal! @a`

Let's dissect that command. 

- `:` enters command mode
- The `%` means select the entire range from the start of the buffer to the end
- `normal` means execute a normal mode command, but that's not enough 
- `normal!` means execute a normal mode command and ignore any remappings. If the user has remapped the `@` command to something else we will ignore that mapping
- `@a` executes the macro

And there it is, an offset added to every number!

11<br />
12<br />
13<br />
14<br />
15<br />
16<br />
17<br />
18<br />
19<br />

I had some help from a Stack Overflow [answer](https://stackoverflow.com/questions/390174/in-vim-how-do-i-apply-a-macro-to-a-set-of-lines), and  I also found a nice [Vim tutorial](http://learnvimscriptthehardway.stevelosh.com/) where I learned about the `normal!` command.
