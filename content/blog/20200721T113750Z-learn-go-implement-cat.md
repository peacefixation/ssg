---
title: "Learning Go by implementing cat"
date: 2020-07-21T21:37:50+10:00
description: "A simple implementation of the core-utils cat command in Go."
tags: [go]
draft: false
---

We've been making the transition from Java to Go and React at the company I work for. Over the last year or so most of our new web applications consist of a Go backend and React front end. This has been a pleasant experience for the most part, but there's a lot to learn so I decided to write a bunch of Go programs on the side to explore the language. After a little thought, it occurred to me that re-writing *nix core utils would be a practical way to go about it. For the most part they are small, single function programs and should be fairly simple to figure out.

As a lightning fast introduction to Go, a simple "hello world" program might look like this:

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello world!")
}
```

The file is saved as `main.go`. It contains a package declaration, `main`, which is the root package name of a Go program. It also contains a function called `main` which is the starting point of the program. I import a package called `fmt` which provides, among other things, a `Println` function that will print a string followed by a new line. You can run this program with the go command `go run main.go`.

The first real program I wrote was `cat`. This program "contatenates files and prints to standard output". I should be able to give it some arguments of the names of the files I would like to use for input, and it should print them one after the other to standard output i.e. `cat file1.txt file2.txt`. I should also be able to pipe input to the program i.e. `echo "foo bar" | cat`. 

Optionally, I would like to set a flag to number the output lines i.e. `cat -n file1.txt file2.txt`.

There's a few key things that I need to implement to achieve this functionality. First, I need to be able to pass arguments to a Go console program. Go has a package called `flag` that handles this in a simple way. You create a flag with a name, a default and a description, and parse the flags that are given to store them in variables:

```go
numberLines := flag.Bool("n", false, "number output lines")
flag.Parse()
```

Then you can check if numberLines (a boolean) is true, and if so the flag to number the lines was given.

```go
if *numberLines {
    // flag was set
}
```

Notice here that I am dereferencing the `numberLines` variable with `*`. The variable is a pointer, and so to get the value of the variable you must dereference the pointer. If you access `numberLines` directly you will find the address of the pointer. Pointers allow us to "pass by reference" instead of "pass by value". This way I can give you the address of a large object instead of a copy of the large object, and you can operate on the object itself instead of a copy of the object.

Any other arguments passed to the program are accessible from the array returned from `flag.Args()`.

The next thing I want to determine is whether my input comes from filename arguments or from a pipe. I can achieve this by calling `info, _ := os.Stdin.Stat()` and then checking the file mode of the returned `info` variable.

```go
info, _ := os.Stdin.Stat()

if (info.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
    // stdin is a character device
} else if info.Size() > 0 {
    // stdin is not a character device and there is data available
}
```

Notice that `os.Stdin.Stat()` returns 2 variables, and I ignore one of them with `_`.

Now I'm ready to handle whatever input is given. I need a function that can take a file and print it to standard output.

```go
func printFile(file *os.File)  {
    reader := bufio.NewReader(file)
    for {
        line, err := reader.ReadString('\n')
        if err == io.EOF {
            fmt.Printf("%s\n", line)
            break
        }

        if err != nil {
            log.Fatalf("Read error: %v\n", err)
        }

        fmt.Printf("%s", line)
    }
}
```

The function is defined with the keyword `func` and arguments with a name first, and then a type (which is the opposite to most other languages I know). `file *os.File` is an argument named `file` of type `*os.File`, a pointer to type `os.File`. This function doesn't return anything.

Inside the function I create a buffered file reader from the `bufio` package and read lines (strings that end with `\n`) until I find the end of the file which is apparent when `reader.ReadString()` returns a specific error `io.EOF`. The idiomatic method of error handling in Go is for functions to return errors for the caller to check.

To handle the optional "number output lines" flag, I enhance this function to keep track of the line number.

```go
func printFile(file *os.File, startLineNumber int) int {
    reader := bufio.NewReader(file)
    for {
        numberStr := ""
        if startLineNumber != -1 {
            numberStr = strconv.Itoa(startLineNumber)
        }

        line, err := reader.ReadString('\n')
        if err == io.EOF {
            fmt.Printf("%8s%s\n", numberStr, line)
            break
        }

        if err != nil {
            log.Fatalf("Read error: %v\n", err)
        }

        fmt.Printf("%8s%s", numberStr, line)
        startLineNumber++
    }

    startLineNumber++ // increment for next file
    return startLineNumber
}
```

The function now has an extra argument `startLineNumber` of type `int` and also returns a value of type `int` which is made clear after the parentheses containing the arguments.

With all of this work done I am ready to handle the filename arguments or the piped data as input to the program and call the `printFile` function to print it to standard output.

```go
func main() {
    numberLines := flag.Bool("n", false, "number output lines")
    flag.Parse()

    info, _ := os.Stdin.Stat()

    startLineNumber := -1
    if *numberLines {
        startLineNumber = 1
    }

    if (info.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
        // were there any filenames as arguments?
        if flag.NArg() < 1 {
            usage()
        }

        // print each file to stdout
        for _, filename := range flag.Args() {
            // open the file
            file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
            if err != nil {
                log.Fatalf("File error: %v\n", err)
            }
            // make sure the file is closed when we're done
            defer file.Close()

            // print the file
            startLineNumber = printFile(file, startLineNumber)
        }
    } else if info.Size() > 0 {
        // print lines piped to standard input
        printFile(os.Stdin, startLineNumber)
    }
}
```

Here I check the file mode of standard input. If the mode is "character device" then I expect filenames as arguments. If there were none, exit with a usage message. Otherwise iterate over each argument with a for loop.

For each filename, I attempt to open the file in read only mode `file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)`. I also call `defer file.Close()`. The defer statement is a way of ensuring that a function is called before the surrounding function returns. By calling `file.Close()` with `defer` I ensure that whatever else happens down the track (other errors perhaps), the file will be closed. If the file is opened without error, I print the file and keep track of the line number so that can be printed as well.

If the file mode of standard input was not "character device" then I check to see if the size of the standard input is greater than zero and if it is I pass the file descriptor for standard input to the `printFile` function for printing.

And there you have it, a simple implementation of `cat` written in Go. You can see the full source at [Github](https://github.com/peacefixation/go-exercises/blob/master/cat/cat.go). I'll write articles about some of my other implementations of core utils in the future.
