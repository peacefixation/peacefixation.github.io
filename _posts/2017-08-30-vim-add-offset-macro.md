---
layout: post
title:  "Add an offset to a list of numbers with Vim"
date:   2017-08-30 21:00:00 +1000
categories: development
---

I had a task today to pull out a bunch of account ids from a log file so we could email the customers. I needed to add an offset to these account ids to turn them into an account number. This is easy enough, but there were a lot of them, and it’s tedious to do by hand. Vim to the rescue!

Given a file containing a list of numbers on consecutive lines, I want to add an offset of 10 to every number.

1 <br />
2 <br />
3 <br />
4 <br />
5 <br />
6 <br />
7 <br />
8 <br />
9 <br />

First a quick Vim refresher. Vim has modes. Normal mode is for navigation and text manipulation, everything you type is interpreted as a command. You can use commands like `gg` to go the start of the file, `G` to go to the end of the file, `$` to go to the end of the line, etc. You can return to normal mode from another mode by pressing `ESC`. Command-line mode is for Vim commands. You enter command-line mode by pressing `:`, then you enter your command. For example `:set nu` turns line numbering on. `:help  command-line-mode` opens the help file for command-line-mode. With that out of the way, we are going to record a macro and then execute it by running a cormal mode command from command-line mode with the `normal` command. Vim is complicated. :)

Create a file with the numbers 1-9 on consequtive lines. Press `ESC` to make sure we're in normal mode. Now we will record a macro called `a`:

- type `qa` to start recording
- type `10` and press `Ctrl-A` to increment the first number by 10
- type `q` to stop recording

You can use `:u` to undo the last command to revert that change, alternatively type `10` and press `Ctrl-X` to decrement the number by 10.

Now we will apply the macro to every line in the file, from normal mode:

- type `:%normal! @a`

Let's dissect that command. 

The `:` enters command mode. `%` means select the entire range from the start of the buffer to the end. `normal` means execute a normal mode command, but that's not enough. `normal!` means execute a normal mode command and ignore any remappings. If the user has remapped the `@` command to something else we will ignore that mapping. `@a` executes the macro named `a`.

And there it is, an offset added to every number!

11 <br />
12 <br />
13 <br />
14 <br />
15 <br />
16 <br />
17 <br />
18 <br />
19 <br />

I had some help from a Stack Overflow [answer](https://stackoverflow.com/questions/390174/in-vim-how-do-i-apply-a-macro-to-a-set-of-lines), and  I also found a nice [Vim tutorial](http://learnvimscriptthehardway.stevelosh.com/) where I learned about the `normal!` command.

Next I had to pull the email addresses out of the database. I included my account numbers in an SQL query with a little help from Vim's substitute command.

- `:%s/$/,/g`

What's this doing? Well `:` enters command mode and `%` selects the whole buffer. Then `s` is the substitute command and we find the end of each line with `$` (which is a zero length match) and replace (well, replacing nothing with something is more like insert) it with a `,` and we specify the `g` (global) flag so we keep replacing after the first match. So now we have a comma after each number, and it's easy to go from this:

11, <br />
12, <br />
13, <br />
14, <br />
15, <br />
16, <br />
17, <br />
18, <br />
19, <br />

to this (which I did manually):

SELECT email_address <br />
FROM the_table <br />
WHERE account_number IN ( <br />
11, <br />
12, <br />
13, <br />
14, <br />
15, <br />
16, <br />
17, <br />
18, <br />
19) <br />

The names of database tables and columns have been changed to protect the innocent, but there you go, that's a tedious, everyday task made quick and easy with the help of an advanced text editor. I'm a big noob with Vim, but here and there I learn a new trick and slowly but surely I get better at it. I hope this helps someone advance a little!