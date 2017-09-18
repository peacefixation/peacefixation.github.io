---
layout: post
title:  "Clean SQL embedded in source files with Vim"
date:   2017-09-11 19:41:00 +1000
categories: development
---

Here's a task I do often, and it's always a 2 step process. There's a program I maintain that queries the database and the SQL is just a long series of concatenated strings in a source file. I copy the query into a text editor and then remove all the quotes and plus characters so I can modify and test it on the database. Ideally SQL would be kept in an SQL file and it would be easier to work with, but you don't always get what you want.

Today I decided that Vim would help me by using the substitute command to clean my SQL. It turned out to be quite simple, the regex is an easy one but Vim has its own ideas about regex syntax. First a quick recap, the substitute command has the structure `:%s/pattern/replacement/flags` where `pattern` is the string to find (or a regex that matches it), `replacement` is the replacement string and `flags` are the regex flags (like `g` for global).

Here's the SQL string that I pulled out of the source file:

    "SELECT a.name, b.number " +
    "FROM table_a a " +
    "JOIN table_b b ON a.b_id = b.a_id " +
    "WHERE a.name LIKE '% Smith' " +
    "AND b.number > 100;"

And the substitute command I came up with to clean it:

    :%s/\(^\s*+\s*"\)\|\(\s*"\s*+\s*$\)//g

Now you might notice that the regex syntax is a little odd. I've used parentheses and alternation to group my expressions but I had to escape the `(`, `)` and `|` characters, and **not** escape the `+` character! This threw me but Vim regex syntax is pretty backwards! You can use the `\v` (magic flag) at the start of the pattern to invert this and I could shorten my command to:

    :%s/\v(^\s*\+?\s*")|(\s*"\s*\+?\s*$)//g

Then I mapped this to a key command `,c` in my `vimrc` so I can run it easily (note the extra `\` escape on the alternation character `|` is required when mapping the command):

    map ,c :%s/\v(^\s*\+?\s*")\|(\s*"\s*\+?\s*$)//g

And there you have it, clean SQL ready to execute:

    SELECT a.name, b.number
    FROM table_a a
    JOIN table_b b ON a.b_id = b.a_id
    WHERE a.name LIKE '% Smith'
    AND b.number > 100;

So it turns out that if you try sometimes, you just might find .. you get what you need! I will chalk up some more Vim experience, and give a little credit to this Stack Overflow [answer](https://vi.stackexchange.com/questions/3115/find-and-replace-using-regular-expressions) and the Vim tips [wiki page](http://vim.wikia.com/wiki/Search_and_replace) for the substitute command.

I've uploaded my fledgling `vimrc` to [Github](https://github.com/peacefixation/vimrc/blob/master/vimrc), may it grow larger and more useful in time!