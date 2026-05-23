---
title: Clean SQL embedded in source files with Vim
date: 2017-09-11T19:41:00+10:00
description: A simple Vim exercise.
tags:
  - programming
  - vim
draft: false
---

There's a task I perform regularly and it's always a 2 step process. One of the programs I maintain queries the database and the SQL is a long series of concatenated strings in a Java source file. I copy the query into a text editor and then remove all the quotes and plus characters so I can modify and test it on the database. Ideally SQL would be kept in an SQL file and it would be easier to work with, but in this case I don't have the liberty of making that modification.

Today I decided that Vim would help me by using the substitute command with a regex to clean my SQL. It turned out to be quite simple, the regex is an easy one but Vim has its own ideas about regex syntax. First a quick recap, the substitute command has the structure `:%s/pattern/replacement/flags` where `pattern` is the string to find (or a regex that matches it), `replacement` is the replacement string and `flags` are the regex flags (like `g` for global).

Here's the SQL string that I pulled out of the source file:

```java
"SELECT a.name, b.number " +
"FROM table_a a " +
"JOIN table_b b ON a.b_id = b.a_id " +
"WHERE a.name LIKE '% Smith' " +
"AND b.number > 100;"
```

And the substitute command I came up with to clean it:

```vim
:%s/\(^\s*+\s*"\)\|\(\s*"\s*+\s*$\)//g
```

Now you might notice that the regex syntax is a little odd. I've used parentheses and alternation to group my expressions but I had to escape the `(`, `)` and `|` characters, and **not** escape the `+` character! This threw me but Vim regex syntax is pretty backwards! You can use the `\v` (magic flag) at the start of the pattern to invert this and I could shorten my command to:

```vim
:%s/\v(^\s*\+?\s*")|(\s*"\s*\+?\s*$)//g
```

I mapped this substition to a key command `,c` in my `vimrc` so I can run it more easily. Note the extra `\` escape on the alternation character `|`. This is required when mapping the command):

```vim
map ,c :%s/\v(^\s*\+?\s*")\|(\s*"\s*\+?\s*$)//g
```

Running the command with `:,c` results in clean SQL ready to execute in the database:

```sql
SELECT a.name, b.number
FROM table_a a
JOIN table_b b ON a.b_id = b.a_id
WHERE a.name LIKE '% Smith'
AND b.number > 100;
```

Credit to this Stack Overflow [answer](https://vi.stackexchange.com/questions/3115/find-and-replace-using-regular-expressions) and the Vim tips [wiki page](http://vim.wikia.com/wiki/Search_and_replace) for the substitute command.

I've uploaded my fledgling `vimrc` to [Github](https://github.com/peacefixation/vimrc/blob/master/vimrc), may it grow larger and more useful in time!
